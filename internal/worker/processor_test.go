package worker_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/queue/queuetest"
	"github.com/ollymarsters/job-scraper/internal/services/identity/identitytest"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
	"github.com/ollymarsters/job-scraper/internal/worker"
	"github.com/ollymarsters/job-scraper/internal/worker/proxy"
	"github.com/ollymarsters/job-scraper/internal/worker/scraper"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

type detailStub struct{ err error }

func (d detailStub) GetDetails(_ context.Context, card dto.Job) (dto.Job, error) {
	return dto.Job{Source: "wis", Title: "Job", URL: card.URL}, d.err
}

const fetchedURL = "https://8.8.8.8/job"

type fetchingDetailer struct{ client *http.Client }

func newFetchingDetailer(cache proxy.Cache) fetchingDetailer {
	upstream := identitytest.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("page"))}, nil
	})
	return fetchingDetailer{client: &http.Client{Transport: proxy.NewFetchTransport(upstream, cache)}}
}

func (d fetchingDetailer) GetDetails(ctx context.Context, card dto.Job) (dto.Job, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchedURL, nil)
	if err != nil {
		return dto.Job{}, err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return dto.Job{}, err
	}
	_ = resp.Body.Close()
	return dto.Job{Source: "wis", Title: "Job", URL: card.URL}, nil
}

type pageSource struct{ next string }

func (pageSource) Cfg() sources.Config { return sources.Config{Name: "wis"} }
func (s pageSource) FetchPage(context.Context, string) ([]dto.Job, string, error) {
	return []dto.Job{{URL: "https://example.com/job/1"}}, s.next, nil
}

type fetchingPage struct {
	client *http.Client
	err    error
}

func (fetchingPage) Cfg() sources.Config { return sources.Config{Name: "fetching"} }
func (s fetchingPage) FetchPage(ctx context.Context, _ string) ([]dto.Job, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchedURL, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	_ = resp.Body.Close()
	return []dto.Job{{URL: "https://example.com/job/1"}}, "", s.err
}

type failingPage struct{ err error }

func (failingPage) Cfg() sources.Config { return sources.Config{Name: "failing"} }
func (s failingPage) FetchPage(context.Context, string) ([]dto.Job, string, error) {
	return nil, "", s.err
}

type blockedOnce struct {
	err     error
	blocked atomic.Bool
}

func (*blockedOnce) Cfg() sources.Config { return sources.Config{Name: "linkedin"} }
func (s *blockedOnce) FetchPage(context.Context, string) ([]dto.Job, string, error) {
	if s.blocked.CompareAndSwap(false, true) {
		return nil, "", s.err
	}
	return []dto.Job{{URL: "https://example.com/job/1"}}, "", nil
}

type allNewURLs struct{}

func (allNewURLs) NewURLs(_ context.Context, urls []string) ([]string, error) { return urls, nil }

type noSearchConfig struct{}

func (noSearchConfig) SearchConfig(context.Context, string) (dto.SearchConfig, error) {
	return dto.SearchConfig{}, data.ErrNotFound
}

type discardCards struct{}

func (discardCards) CapturePage(context.Context, dto.SourceTarget, []dto.Job, dto.SearchConfig) error {
	return nil
}

type oneBoardJob struct{}

func (oneBoardJob) FetchBoard(context.Context, dto.BoardPoll) (sources.BoardResult, error) {
	return sources.BoardResult{Jobs: []dto.Job{{Title: "Engineer", URL: "https://example.com/1"}}}, nil
}

type ingest struct {
	server   *httptest.Server
	requests atomic.Int32
	lastJob  atomic.Pointer[dto.Job]
}

func newIngest(t *testing.T, status int) *ingest {
	t.Helper()
	in := &ingest{}
	in.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		in.requests.Add(1)
		var job dto.Job
		if json.NewDecoder(r.Body).Decode(&job) == nil {
			in.lastJob.Store(&job)
		}
		w.WriteHeader(status)
		if strings.HasSuffix(r.URL.Path, "/batch") {
			_, _ = w.Write([]byte(`{"results":[{"status":"new"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"new"}`))
	}))
	t.Cleanup(in.server.Close)
	return in
}

type fixture struct {
	store     *jobsearchtest.FakeStore
	processor *worker.Processor
	ingest    *ingest
	published *queuetest.Recorder
}

func newFixture(t *testing.T, ingestStatus int, nextCursor string) fixture {
	t.Helper()
	return newCappedFixture(t, ingestStatus, nextCursor, 0)
}

func newCappedFixture(t *testing.T, ingestStatus int, nextCursor string, maxPages int) fixture {
	t.Helper()
	store := jobsearchtest.NewFakeStore()
	in := newIngest(t, ingestStatus)
	exporter := scraper.NewAPIExporter(in.server.URL, "token").WithInitialBackoff(0)
	published := queuetest.NewRecorder()
	page := fetchingPage{client: newFetchingDetailer(store).client}
	linkedin := &blockedOnce{err: &sources.DeferredError{Until: time.Now().Add(10 * time.Minute)}}
	build := func(target dto.SourceTarget) (sources.Source, bool) {
		switch target.Source {
		case "fetching":
			return page, true
		case "indeed":
			return failingPage{err: fmt.Errorf("indeed: %w", sources.ErrSourceKeyRejected)}, true
		case "unavailable":
			return failingPage{err: &sources.StatusError{Code: http.StatusInternalServerError, Status: "500"}}, true
		case "limited":
			return failingPage{err: &sources.StatusError{Code: http.StatusTooManyRequests, Status: "429", RetryAfter: time.Minute}}, true
		case "linkedin":
			return linkedin, true
		case "fetchfail":
			page.err = errors.New("page failed after fetch")
			return page, true
		}
		return pageSource{next: nextCursor}, true
	}
	processor := worker.NewProcessor(worker.Deps{
		JS:           jobsearch.Build(jobsearchtest.NewDeps(store)),
		Broker:       published,
		Orchestrator: scraper.New(allNewURLs{}, noSearchConfig{}, build, discardCards{}),
		Boards:       scraper.NewBoardPoller(store, oneBoardJob{}, exporter),
		Exporter:     exporter,
		MaxPages:     maxPages,
		Verify: func(ctx context.Context, source, token string) (scraper.Verification, error) {
			return scraper.VerifyBoard(ctx, build, source, token)
		},
		CardComplete: map[string]bool{"remoteok": true},
		Detailers: map[string]sources.DetailFetcher{
			"wis":      detailStub{},
			"indeed":   newFetchingDetailer(store),
			"linkedin": detailStub{err: errors.New("page gone")},
			"gone":     detailStub{err: fmt.Errorf("%w: status 404", sources.ErrGone)},
			"blocked":  detailStub{err: &sources.DeferredError{Until: time.Now().Add(10 * time.Minute)}},
		},
	})
	return fixture{store: store, processor: processor, ingest: in, published: published}
}

func (f fixture) discoverProcessor(discover worker.DiscoverFunc, scoring worker.IncludeFilterConfigs) *worker.Processor {
	exporter := scraper.NewAPIExporter(f.ingest.server.URL, "token").WithInitialBackoff(0)
	return worker.NewProcessor(worker.Deps{
		JS:       jobsearch.Build(jobsearchtest.NewDeps(f.store)),
		Boards:   scraper.NewBoardPoller(f.store, oneBoardJob{}, exporter),
		Broker:   f.published,
		Scoring:  scoring,
		Discover: discover,
	})
}

func (f fixture) target(t *testing.T, source string) dto.SourceTarget {
	t.Helper()
	target, err := f.store.CreateSourceTargetWithRun(t.Context(), "user-1", source, "https://example.com/search", true, nil, dto.RunWindow{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return target
}

func (f fixture) runStatus(t *testing.T, target dto.SourceTarget) dto.SourceTarget {
	t.Helper()
	got, err := f.store.GetSourceTarget(t.Context(), target.ID)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func (f fixture) board(t *testing.T) string {
	t.Helper()
	company, err := f.store.UpsertCompany(t.Context(), dto.CompanyUpsert{Slug: "acme", Name: "Acme"})
	if err != nil {
		t.Fatal(err)
	}
	board, err := f.store.UpsertCandidateBoard(t.Context(), company.ID, "greenhouse", "acme")
	if err != nil {
		t.Fatal(err)
	}
	return board.ID
}

func TestProcess(t *testing.T) {
	ctx := t.Context()

	exports := []struct {
		name string
		task queue.Task
	}{
		{"detail task exports the fetched job", queuetest.DetailTask("wis")},
		{"full feed card exports without a detail fetcher", queuetest.DetailTask("remoteok")},
	}
	for _, tt := range exports {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, http.StatusOK, "")
			if err := f.processor.Process(ctx, tt.task); err != nil {
				t.Fatal(err)
			}
			if got := f.ingest.requests.Load(); got != 1 {
				t.Fatalf("ingest requests = %d, want 1", got)
			}
		})
	}

	t.Run("detail task exports the card's posted date", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		postedAt := time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)
		task := queuetest.DetailTask("wis")
		task.Card.PostedAt = &postedAt
		if err := f.processor.Process(ctx, task); err != nil {
			t.Fatal(err)
		}
		job := f.ingest.lastJob.Load()
		if job == nil || job.PostedAt == nil || !job.PostedAt.Equal(postedAt) {
			t.Fatalf("exported job = %+v, want PostedAt %v", job, postedAt)
		}
	})

	t.Run("gone detail is acked without exporting", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		if err := f.processor.Process(ctx, queuetest.DetailTask("gone")); err != nil {
			t.Fatalf("Process() = %v, want nil so the task is acked", err)
		}
		if got := f.ingest.requests.Load(); got != 0 {
			t.Fatalf("ingest requests = %d, want 0", got)
		}
	})

	fails := []struct {
		name           string
		task           queue.Task
		ingestStatus   int
		wantIngestHits int32
	}{
		{"unsupported task kind fails without exporting", queue.Task{Source: "wis", Kind: "bogus"}, http.StatusOK, 0},
		{"source without a detail fetcher fails without exporting", queuetest.DetailTask("bogus"), http.StatusOK, 0},
		{"detail fetch failure is returned for requeue", queuetest.DetailTask("linkedin"), http.StatusOK, 0},
		{"transient ingest failure is returned for requeue", queuetest.DetailTask("wis"), http.StatusServiceUnavailable, 3},
	}
	for _, tt := range fails {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, tt.ingestStatus, "")
			if err := f.processor.Process(ctx, tt.task); err == nil {
				t.Fatal("Process() = nil, want error so the task is not acked")
			}
			if got := f.ingest.requests.Load(); got != tt.wantIngestHits {
				t.Fatalf("ingest requests = %d, want %d", got, tt.wantIngestHits)
			}
		})
	}

	t.Run("detail task forgets its cached fetches once exported", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		if err := f.processor.Process(ctx, queuetest.DetailTask("indeed")); err != nil {
			t.Fatal(err)
		}
		if _, ok, _ := f.store.LookupFetch(ctx, fetchedURL); ok {
			t.Fatal("cached fetch still present after a successful task")
		}
	})

	t.Run("detail task keeps its cached fetches when the export fails", func(t *testing.T) {
		f := newFixture(t, http.StatusServiceUnavailable, "")
		if err := f.processor.Process(ctx, queuetest.DetailTask("indeed")); err == nil {
			t.Fatal("Process() = nil, want export error")
		}
		if _, ok, _ := f.store.LookupFetch(ctx, fetchedURL); !ok {
			t.Fatal("cached fetch missing after a failed task, want it kept for the retry")
		}
	})

	t.Run("listing page task forgets its cached fetches once it succeeds", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		target := f.target(t, "fetching")
		task := queuetest.ListingTask("fetching")
		task.TargetID, task.RunID = target.ID, target.RunID
		if err := f.processor.Process(ctx, task); err != nil {
			t.Fatal(err)
		}
		if _, ok, _ := f.store.LookupFetch(ctx, fetchedURL); ok {
			t.Fatal("cached fetch still present after a successful page")
		}
	})

	t.Run("listing page task keeps its cached fetches when it fails after fetching", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		target := f.target(t, "fetchfail")
		task := queuetest.ListingTask("fetchfail")
		task.TargetID, task.RunID = target.ID, target.RunID
		if err := f.processor.Process(ctx, task); err == nil {
			t.Fatal("Process() = nil, want page error")
		}
		if _, ok, _ := f.store.LookupFetch(ctx, fetchedURL); !ok {
			t.Fatal("cached fetch missing after a failed page, want it kept for the retry")
		}
	})

	t.Run("redelivered page task for a finished run does not scrape again", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		target := f.target(t, "wis")
		if _, err := f.store.TransitionSourceTargetRun(ctx, target.ID, target.RunID, "succeeded", ""); err != nil {
			t.Fatal(err)
		}
		task := queuetest.ListingTask("wis")
		task.TargetID, task.RunID, task.Redelivered = target.ID, target.RunID, true
		if err := f.processor.Process(ctx, task); err != nil {
			t.Fatal(err)
		}
		if got := f.runStatus(t, target).RunStatus; got != "succeeded" || len(f.published.Tasks()) != 0 {
			t.Fatalf("run status = %s, published = %d, want succeeded and none", got, len(f.published.Tasks()))
		}
	})

	t.Run("first delivery of the last listing page succeeds the run", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		target := f.target(t, "wis")
		task := queuetest.ListingTask("wis")
		task.TargetID, task.RunID = target.ID, target.RunID
		if err := f.processor.Process(ctx, task); err != nil {
			t.Fatal(err)
		}
		if got := f.runStatus(t, target).RunStatus; got != "succeeded" {
			t.Fatalf("run status = %s, want succeeded", got)
		}
	})

	t.Run("first delivery of a listing page with more pages publishes the next cursor", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "2:5")
		target := f.target(t, "wis")
		task := queuetest.ListingTask("wis")
		task.TargetID, task.RunID = target.ID, target.RunID
		if err := f.processor.Process(ctx, task); err != nil {
			t.Fatal(err)
		}
		published := f.published.Tasks()
		if len(published) != 1 || published[0].Cursor != "2:5" || published[0].ID == task.ID {
			t.Fatalf("published = %+v, want one task with cursor 2:5 and a new ID", published)
		}
		if got := f.runStatus(t, target).RunStatus; got != "running" {
			t.Fatalf("run status = %s, want running", got)
		}
	})

	t.Run("listing page under the page cap publishes the next page", func(t *testing.T) {
		f := newCappedFixture(t, http.StatusOK, "2:5", 2)
		target := f.target(t, "wis")
		task := queuetest.ListingTask("wis")
		task.TargetID, task.RunID = target.ID, target.RunID
		if err := f.processor.Process(ctx, task); err != nil {
			t.Fatal(err)
		}
		published := f.published.Tasks()
		if len(published) != 1 || published[0].Page != 1 {
			t.Fatalf("published = %+v, want one task on page 1", published)
		}
	})

	t.Run("listing page at the page cap succeeds the run without publishing", func(t *testing.T) {
		f := newCappedFixture(t, http.StatusOK, "2:5", 2)
		target := f.target(t, "wis")
		task := queuetest.ListingTask("wis")
		task.TargetID, task.RunID, task.Cursor, task.Page = target.ID, target.RunID, "1:5", 1
		if err := f.processor.Process(ctx, task); err != nil {
			t.Fatal(err)
		}
		if got := f.runStatus(t, target).RunStatus; got != "succeeded" || len(f.published.Tasks()) != 0 {
			t.Fatalf("run status = %s, published = %d, want succeeded and none", got, len(f.published.Tasks()))
		}
	})

	t.Run("board check polls the board and succeeds its run", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		target := f.target(t, "greenhouse")
		task := queue.Task{Version: 1, Source: "greenhouse", Kind: queue.BoardCheckTask, BoardID: f.board(t), TargetID: target.ID, RunID: target.RunID, Manual: true}
		if err := f.processor.Process(ctx, task); err != nil {
			t.Fatal(err)
		}
		if got := f.ingest.requests.Load(); got != 1 {
			t.Fatalf("ingest requests = %d, want 1", got)
		}
		if got := f.runStatus(t, target).RunStatus; got != "succeeded" {
			t.Fatalf("run status = %s, want succeeded", got)
		}
	})

	t.Run("board check for a missing board is acked when not manual", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		task := queue.Task{Version: 1, Source: "greenhouse", Kind: queue.BoardCheckTask, BoardID: "gone"}
		if err := f.processor.Process(ctx, task); err != nil {
			t.Fatalf("Process() = %v, want nil", err)
		}
		if got := f.ingest.requests.Load(); got != 0 {
			t.Fatalf("ingest requests = %d, want 0", got)
		}
	})

	t.Run("board verify with an unusable token is acked without verifying", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		company, err := f.store.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.store.UpsertCandidateBoard(ctx, company.ID, "greenhouse", "a/b"); err != nil {
			t.Fatal(err)
		}
		task := queue.Task{Version: 1, Source: "greenhouse", Kind: queue.BoardVerifyTask, CompanyID: company.ID, BoardToken: "a/b"}
		if err := f.processor.Process(ctx, task); err != nil {
			t.Fatalf("Process() = %v, want nil", err)
		}
		boards, err := f.store.ListCompanyBoards(ctx, company.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, board := range boards {
			if board.Status == dto.BoardVerified {
				t.Fatalf("board %+v verified despite failed verification", board)
			}
		}
	})

	for _, tt := range []struct {
		name         string
		poll         bool
		wantIngested int32
	}{
		{name: "board verify without poll only verifies", poll: false, wantIngested: 0},
		{name: "board verify with poll verifies then polls the board", poll: true, wantIngested: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, http.StatusOK, "")
			company, err := f.store.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme"})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.store.UpsertCandidateBoard(ctx, company.ID, "greenhouse", "acme"); err != nil {
				t.Fatal(err)
			}
			task := queue.Task{Version: 1, Source: "greenhouse", Kind: queue.BoardVerifyTask, CompanyID: company.ID, BoardToken: "acme", Poll: tt.poll}
			if err := f.processor.Process(ctx, task); err != nil {
				t.Fatalf("Process() = %v, want nil", err)
			}
			boards, err := f.store.ListCompanyBoards(ctx, company.ID)
			if err != nil {
				t.Fatal(err)
			}
			if len(boards) != 1 || boards[0].Status != dto.BoardVerified {
				t.Fatalf("boards = %+v, want one verified board", boards)
			}
			if got := f.ingest.requests.Load(); got != tt.wantIngested {
				t.Fatalf("ingest requests = %d, want %d", got, tt.wantIngested)
			}
		})
	}
}

type configsStub []dto.SearchConfig

func (c configsStub) IncludeFilterConfigs(context.Context) ([]dto.SearchConfig, error) { return c, nil }

func TestProcessBoardDiscover(t *testing.T) {
	ctx := t.Context()
	jobs := []dto.Job{{Title: "Go Engineer", Location: "London"}}
	discover := func(context.Context, string, string) (scraper.Discovery, error) {
		return scraper.Discovery{Name: "Acme Corp", Jobs: jobs, Recheck: true}, nil
	}
	task := queue.Task{Version: 1, Source: "greenhouse", Kind: queue.BoardDiscoverTask, BoardToken: "acme"}
	match := dto.SearchConfig{UserID: "match", RequiredTitleKeywords: []string{"go"}}
	miss := dto.SearchConfig{UserID: "miss", RequiredTitleKeywords: []string{"rust"}}
	prior := dto.SearchConfig{UserID: "prior", RequiredTitleKeywords: []string{"go"}}

	t.Run("tracks matching users as new and leaves existing rows alone", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		company, err := f.store.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme Corp"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.store.SetCompanyTracking(ctx, "prior", company.ID, true, 0); err != nil {
			t.Fatal(err)
		}
		if _, err := f.store.SetCompanyReviewState(ctx, "prior", company.ID, "dismissed"); err != nil {
			t.Fatal(err)
		}
		p := f.discoverProcessor(discover, configsStub{match, miss, prior})
		if err := p.Process(ctx, task); err != nil {
			t.Fatalf("Process() = %v, want nil", err)
		}
		boards, err := f.store.ListCompanyBoards(ctx, company.ID)
		if err != nil || len(boards) != 1 || boards[0].Status != dto.BoardVerified || boards[0].VerificationMethod != "discovered" {
			t.Fatalf("ListCompanyBoards(%s) = %+v, %v, want one board verified as discovered", company.ID, boards, err)
		}
		for user, want := range map[string]string{"match": "new", "prior": "dismissed", "miss": ""} {
			tracked, err := f.store.ListTrackedCompaniesForUser(ctx, user)
			if err != nil {
				t.Fatal(err)
			}
			got := ""
			if len(tracked) == 1 {
				got = tracked[0].ReviewState
			}
			if got != want {
				t.Errorf("review state for %s = %q, want %q", user, got, want)
			}
		}
	})

	t.Run("a tracked board's fetched jobs are ingested in the same task", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		if err := f.discoverProcessor(discover, configsStub{match}).Process(ctx, task); err != nil {
			t.Fatalf("Process() = %v, want nil", err)
		}
		if got := f.ingest.requests.Load(); got != 1 {
			t.Errorf("ingest requests = %d, want 1", got)
		}
		due, err := f.store.ListDueBoards(ctx)
		if err != nil || len(due) != 1 {
			t.Fatalf("ListDueBoards() = %+v, %v, want the completed board released", due, err)
		}
	})

	t.Run("a tracked board's discovered profile is saved", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		withProfile := func(context.Context, string, string) (scraper.Discovery, error) {
			return scraper.Discovery{Name: "Acme Corp", Jobs: jobs, Recheck: true, Profile: &dto.CompanyProfile{HQ: "London"}}, nil
		}
		if err := f.discoverProcessor(withProfile, configsStub{match}).Process(ctx, task); err != nil {
			t.Fatalf("Process() = %v, want nil", err)
		}
		got, err := f.store.ListNewCompanies(ctx, "match")
		if err != nil || len(got) != 1 || got[0].Profile == nil || got[0].Profile.HQ != "London" {
			t.Fatalf("ListNewCompanies() = %+v, %v, want one company with the London profile", got, err)
		}
	})

	t.Run("a task company attaches the board without upserting by name", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		company, err := f.store.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "harvest-slug", Name: "Harvest Slug"})
		if err != nil {
			t.Fatal(err)
		}
		withCompany := task
		withCompany.CompanyID = company.ID
		if err := f.discoverProcessor(discover, configsStub{match}).Process(ctx, withCompany); err != nil {
			t.Fatalf("Process() = %v, want nil", err)
		}
		tracked, err := f.store.ListTrackedCompaniesForUser(ctx, "match")
		if err != nil || len(tracked) != 1 || tracked[0].ID != company.ID {
			t.Fatalf("ListTrackedCompaniesForUser(match) = %+v, %v, want only the task's company %s", tracked, err, company.ID)
		}
	})

	t.Run("rediscovery reuses the company already owning the board", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		company, err := f.store.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "harvest-slug", Name: "Harvest Slug"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.store.UpsertCandidateBoard(ctx, company.ID, "greenhouse", "acme"); err != nil {
			t.Fatal(err)
		}
		if err := f.discoverProcessor(discover, configsStub{match}).Process(ctx, task); err != nil {
			t.Fatalf("Process() = %v, want nil", err)
		}
		tracked, err := f.store.ListTrackedCompaniesForUser(ctx, "match")
		if err != nil || len(tracked) != 1 || tracked[0].ID != company.ID {
			t.Fatalf("ListTrackedCompaniesForUser(match) = %+v, %v, want the owning company %s", tracked, err, company.ID)
		}
	})

	t.Run("a board matching nobody stays verified and untracked", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		p := f.discoverProcessor(discover, configsStub{miss})
		if err := p.Process(ctx, task); err != nil {
			t.Fatalf("Process() = %v, want nil", err)
		}
		if got := f.ingest.requests.Load(); got != 0 {
			t.Errorf("ingest requests = %d, want 0", got)
		}
		tracked, err := f.store.ListTrackedCompaniesForUser(ctx, "miss")
		if err != nil || len(tracked) != 0 {
			t.Fatalf("ListTrackedCompaniesForUser(miss) = %+v, %v, want none", tracked, err)
		}
	})

	t.Run("a required-UK task with no jobs or a failed fetch records a miss and is not verified", func(t *testing.T) {
		t.Parallel()
		empty := func(context.Context, string, string) (scraper.Discovery, error) {
			return scraper.Discovery{Name: "Acme Corp"}, nil
		}
		failing := func(context.Context, string, string) (scraper.Discovery, error) {
			return scraper.Discovery{}, errors.New("boom")
		}
		for name, fn := range map[string]worker.DiscoverFunc{"no jobs": empty, "failed fetch": failing} {
			f := newFixture(t, http.StatusOK, "")
			uk := task
			uk.RequireUK = true
			if err := f.discoverProcessor(fn, configsStub{match}).Process(ctx, uk); err != nil {
				t.Fatalf("%s: Process() = %v, want nil", name, err)
			}
			missed, err := f.store.DiscoverMissedSince(ctx, "greenhouse", "acme", time.Now().Add(-time.Minute))
			if err != nil || !missed {
				t.Errorf("%s: DiscoverMissedSince = %v, %v, want true", name, missed, err)
			}
			if _, err := f.store.GetVerifiedBoardID(ctx, "greenhouse", "acme"); !errors.Is(err, data.ErrNotFound) {
				t.Errorf("%s: GetVerifiedBoardID err = %v, want ErrNotFound", name, err)
			}
		}
	})

	t.Run("UK gate", func(t *testing.T) {
		t.Parallel()
		nonUK := func(context.Context, string, string) (scraper.Discovery, error) {
			return scraper.Discovery{Name: "Acme Corp", Recheck: true, Jobs: []dto.Job{{Title: "Go Engineer", Location: "Berlin"}}}, nil
		}
		for name, tc := range map[string]struct {
			fn          worker.DiscoverFunc
			requireUK   bool
			wantTracked int
			wantStamped bool
		}{
			"required-UK non-UK jobs verify but track nobody": {nonUK, true, 0, false},
			"ingest-queued non-UK jobs are tracked":           {nonUK, false, 1, false},
			"required-UK UK jobs are stamped and tracked":     {discover, true, 1, true},
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				f := newFixture(t, http.StatusOK, "")
				tk := task
				tk.RequireUK = tc.requireUK
				if err := f.discoverProcessor(tc.fn, configsStub{match}).Process(ctx, tk); err != nil {
					t.Fatalf("Process() = %v, want nil", err)
				}
				tracked, err := f.store.ListTrackedCompaniesForUser(ctx, "match")
				if err != nil || len(tracked) != tc.wantTracked {
					t.Errorf("ListTrackedCompaniesForUser(match) = %+v, %v, want %d", tracked, err, tc.wantTracked)
				}
				company, err := f.store.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme Corp"})
				if err != nil {
					t.Fatal(err)
				}
				boards, err := f.store.ListCompanyBoards(ctx, company.ID)
				if err != nil || len(boards) != 1 {
					t.Fatalf("ListCompanyBoards() = %+v, %v, want one board", boards, err)
				}
				if boards[0].Status != dto.BoardVerified {
					t.Errorf("board status = %q, want verified", boards[0].Status)
				}
				if got := boards[0].UKJobsSeenAt != nil; got != tc.wantStamped {
					t.Errorf("UKJobsSeenAt set = %v, want %v", got, tc.wantStamped)
				}
			})
		}
	})

	t.Run("an ingest-queued task with no jobs records no miss and still verifies", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t, http.StatusOK, "")
		empty := func(context.Context, string, string) (scraper.Discovery, error) {
			return scraper.Discovery{Name: "Acme Corp"}, nil
		}
		if err := f.discoverProcessor(empty, configsStub{match}).Process(ctx, task); err != nil {
			t.Fatalf("Process() = %v, want nil", err)
		}
		if missed, _ := f.store.DiscoverMissedSince(ctx, "greenhouse", "acme", time.Now().Add(-time.Minute)); missed {
			t.Error("DiscoverMissedSince = true, want false")
		}
		if _, err := f.store.GetVerifiedBoardID(ctx, "greenhouse", "acme"); err != nil {
			t.Errorf("GetVerifiedBoardID err = %v, want verified", err)
		}
	})

	t.Run("a board that is not worth rechecking is verified wttj-origin", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		noRecheck := func(context.Context, string, string) (scraper.Discovery, error) {
			return scraper.Discovery{Name: "Acme Corp"}, nil
		}
		if err := f.discoverProcessor(noRecheck, configsStub{match}).Process(ctx, queue.Task{Version: 1, Source: "wttj", Kind: queue.BoardDiscoverTask, BoardToken: "acme"}); err != nil {
			t.Fatalf("Process() = %v, want nil", err)
		}
		company, err := f.store.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme Corp"})
		if err != nil {
			t.Fatal(err)
		}
		boards, err := f.store.ListCompanyBoards(ctx, company.ID)
		if err != nil || len(boards) != 1 || boards[0].VerificationMethod != scraper.MethodWTTJOrigin {
			t.Fatalf("ListCompanyBoards(%s) = %+v, %v, want one board verified wttj-origin", company.ID, boards, err)
		}
		untracked, err := f.store.ListUntrackedDiscoveredBoards(ctx, false, time.Now())
		if err != nil || len(untracked) != 0 {
			t.Fatalf("ListUntrackedDiscoveredBoards() = %+v, %v, want none", untracked, err)
		}
	})

	t.Run("apply urls on another ATS pivot to its board instead of tracking", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		ashbyJobs := []dto.Job{{Title: "Go Engineer", Location: "London", ApplyURL: "https://jobs.ashbyhq.com/acme/123"}}
		viaAshby := func(context.Context, string, string) (scraper.Discovery, error) {
			return scraper.Discovery{Name: "Acme Corp", Jobs: ashbyJobs}, nil
		}
		wttj := queue.Task{Version: 1, Source: "wttj", Kind: queue.BoardDiscoverTask, BoardToken: "acme"}
		if err := f.discoverProcessor(viaAshby, configsStub{match}).Process(ctx, wttj); err != nil {
			t.Fatalf("Process() = %v, want nil", err)
		}
		company, err := f.store.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme Corp"})
		if err != nil {
			t.Fatal(err)
		}
		tasks := f.published.Tasks()
		if len(tasks) != 1 || tasks[0].Kind != queue.BoardDiscoverTask || tasks[0].Source != "ashby" || tasks[0].BoardToken != "acme" || tasks[0].CompanyID != company.ID || tasks[0].Via != "wttj" {
			t.Fatalf("published = %+v, want one ashby/acme discover for company %s", tasks, company.ID)
		}
		tracked, err := f.store.ListTrackedCompaniesForUser(ctx, "match")
		if err != nil || len(tracked) != 0 {
			t.Fatalf("ListTrackedCompaniesForUser(match) = %+v, %v, want none", tracked, err)
		}
		if got := f.ingest.requests.Load(); got != 0 {
			t.Errorf("ingest requests = %d, want 0", got)
		}
	})

	t.Run("an already verified ATS board is not republished", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		company, err := f.store.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme Corp"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.store.UpsertCandidateBoard(ctx, company.ID, "ashby", "acme"); err != nil {
			t.Fatal(err)
		}
		if _, err := f.store.VerifyCompanyBoard(ctx, company.ID, "ashby", "acme", "discovered", ""); err != nil {
			t.Fatal(err)
		}
		viaAshby := func(context.Context, string, string) (scraper.Discovery, error) {
			return scraper.Discovery{Name: "Acme Corp", Jobs: []dto.Job{{Title: "Go Engineer", ApplyURL: "https://jobs.ashbyhq.com/acme/123"}}}, nil
		}
		wttj := queue.Task{Version: 1, Source: "wttj", Kind: queue.BoardDiscoverTask, BoardToken: "acme"}
		if err := f.discoverProcessor(viaAshby, configsStub{match}).Process(ctx, wttj); err != nil {
			t.Fatalf("Process() = %v, want nil", err)
		}
		if tasks := f.published.Tasks(); len(tasks) != 0 {
			t.Errorf("published = %+v, want none", tasks)
		}
	})

	t.Run("an unresolvable apply host falls through to tracking", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		viaWorkday := func(context.Context, string, string) (scraper.Discovery, error) {
			return scraper.Discovery{Name: "Acme Corp", Jobs: []dto.Job{{Title: "Go Engineer", Location: "London", ApplyURL: "https://acme.wd3.myworkdayjobs.com/en/1"}}}, nil
		}
		wttj := queue.Task{Version: 1, Source: "wttj", Kind: queue.BoardDiscoverTask, BoardToken: "acme"}
		if err := f.discoverProcessor(viaWorkday, configsStub{match}).Process(ctx, wttj); err != nil {
			t.Fatalf("Process() = %v, want nil", err)
		}
		tracked, err := f.store.ListTrackedCompaniesForUser(ctx, "match")
		if err != nil || len(tracked) != 1 {
			t.Fatalf("ListTrackedCompaniesForUser(match) = %+v, %v, want one", tracked, err)
		}
		if tasks := f.published.Tasks(); len(tasks) != 0 {
			t.Errorf("published = %+v, want none", tasks)
		}
	})

	t.Run("a failed fetch is acked without creating a company", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		failing := func(context.Context, string, string) (scraper.Discovery, error) {
			return scraper.Discovery{}, errors.New("board gone")
		}
		if err := f.discoverProcessor(failing, configsStub{match}).Process(ctx, task); err != nil {
			t.Fatalf("Process() = %v, want nil", err)
		}
		page, err := f.store.PageCompaniesForUser(ctx, "match", dto.CompanyPageOptions{Limit: 100})
		if err != nil || len(page.Items) != 0 {
			t.Fatalf("PageCompaniesForUser() = %+v, %v, want none", page, err)
		}
	})
}

func TestProcessKeyRejection(t *testing.T) {
	ctx := t.Context()
	scrape := func(t *testing.T, f fixture, source string) (dto.SourceTarget, error) {
		t.Helper()
		target := f.target(t, source)
		task := queuetest.ListingTask(source)
		task.TargetID, task.RunID = target.ID, target.RunID
		return target, f.processor.Process(ctx, task)
	}

	t.Run("a rejected key fails the run and disables every target of the source", func(t *testing.T) {
		var logs bytes.Buffer
		prev := slog.Default()
		slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
		t.Cleanup(func() { slog.SetDefault(prev) })
		f := newFixture(t, http.StatusOK, "")
		other, err := f.store.CreateSourceTarget(ctx, "user-2", "indeed", "rust", true, nil, dto.RunWindow{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		unrelated := f.target(t, "wis")

		target, err := scrape(t, f, "indeed")
		if err != nil {
			t.Fatalf("Process() = %v, want nil", err)
		}

		got := f.runStatus(t, target)
		if got.Enabled || got.DisabledReason != "indeed key rejected" || got.RunStatus != "failed" || got.LastRunError != "indeed key rejected" {
			t.Errorf("rejected target = %+v, want disabled, failed and reason set", got)
		}
		if got := f.runStatus(t, other); got.Enabled || got.DisabledReason != "indeed key rejected" {
			t.Errorf("other user's indeed target = %+v, want disabled with reason", got)
		}
		if got := f.runStatus(t, unrelated); !got.Enabled || got.DisabledReason != "" {
			t.Errorf("unrelated target = %+v, want untouched", got)
		}
		if n := strings.Count(logs.String(), `"level":"ERROR"`); n != 1 {
			t.Errorf("ERROR log lines = %d, want 1\n%s", n, logs.String())
		}
	})

	t.Run("a non-auth failure leaves targets enabled and retries", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		target, err := scrape(t, f, "unavailable")
		if err == nil {
			t.Fatal("Process() = nil, want the fetch error")
		}
		if got := f.runStatus(t, target); !got.Enabled || got.DisabledReason != "" {
			t.Errorf("target = %+v, want still enabled", got)
		}
	})
}

func TestFailRun(t *testing.T) {
	ctx := t.Context()

	t.Run("marks the run failed", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		target := f.target(t, "wis")
		task := queue.Task{Source: "wis", Kind: queue.ListingPageTask, TargetID: target.ID, RunID: target.RunID}
		if err := f.processor.FailRun(ctx, task); err != nil {
			t.Fatal(err)
		}
		got := f.runStatus(t, target)
		if got.RunStatus != "failed" || got.LastRunError == "" {
			t.Fatalf("run status = %q, error = %q", got.RunStatus, got.LastRunError)
		}
	})

	t.Run("detail task leaves runs alone", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		target := f.target(t, "wis")
		task := queue.Task{Source: "wis", Kind: queue.DetailTask, TargetID: target.ID, RunID: target.RunID}
		if err := f.processor.FailRun(ctx, task); err != nil {
			t.Fatal(err)
		}
		if got := f.runStatus(t, target).RunStatus; got != "queued" {
			t.Fatalf("run status = %q, want queued", got)
		}
	})

	t.Run("a dead-lettered scheduled board check makes the board due again", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t, http.StatusOK, "")
		company, err := f.store.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme"})
		if err != nil {
			t.Fatal(err)
		}
		board, err := f.store.UpsertCandidateBoard(ctx, company.ID, "greenhouse", "acme")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.store.VerifyCompanyBoard(ctx, company.ID, "greenhouse", "acme", "discovered", ""); err != nil {
			t.Fatal(err)
		}
		if err := f.store.MarkBoardCheckPublished(ctx, board.ID); err != nil {
			t.Fatal(err)
		}
		task := queue.Task{Version: 1, Source: "greenhouse", Kind: queue.BoardCheckTask, BoardID: board.ID}
		if err := f.processor.FailRun(ctx, task); err != nil {
			t.Fatal(err)
		}
		due, err := f.store.ListDueBoards(ctx)
		if err != nil || len(due) != 1 || due[0].ID != board.ID {
			t.Fatalf("ListDueBoards() = %+v, %v, want %s due again", due, err, board.ID)
		}
	})

	t.Run("run already gone is not an error", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		task := queue.Task{Source: "wis", Kind: queue.ListingPageTask, TargetID: "gone", RunID: "gone"}
		if err := f.processor.FailRun(ctx, task); err != nil {
			t.Fatal(err)
		}
	})
}

type failingBoard struct{ err error }

func (f failingBoard) FetchBoard(context.Context, dto.BoardPoll) (sources.BoardResult, error) {
	return sources.BoardResult{}, f.err
}

func TestProcessRateLimit(t *testing.T) {
	ctx := t.Context()
	limited := &sources.StatusError{Code: http.StatusTooManyRequests, Status: "429", RetryAfter: time.Minute}
	blocked := &sources.DeferredError{Until: time.Now().Add(10 * time.Minute)}

	discoverTask := queue.Task{Version: 1, Source: "workable", Kind: queue.BoardDiscoverTask, BoardToken: "acme"}
	discovers := []struct {
		name      string
		err       error
		wantDefer bool
	}{
		{"a 429 defers discovery", limited, true},
		{"a blocked gate defers discovery", blocked, true},
		{"any other failure drops the discovery", errors.New("boom"), false},
	}
	for _, tt := range discovers {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, http.StatusOK, "")
			p := f.discoverProcessor(func(context.Context, string, string) (scraper.Discovery, error) {
				return scraper.Discovery{}, tt.err
			}, configsStub{})
			err := p.Process(ctx, discoverTask)
			var deferred *queue.DeferError
			if got := errors.As(err, &deferred); got != tt.wantDefer {
				t.Fatalf("Process() = %v, want deferral %v", err, tt.wantDefer)
			}
			if !tt.wantDefer && err != nil {
				t.Errorf("Process() = %v, want nil so the task is acked", err)
			}
			if tt.wantDefer && !deferred.Until.After(time.Now()) {
				t.Errorf("deferred until %v, want a future time", deferred.Until)
			}
		})
	}

	listings := []struct {
		name   string
		source string
		want   time.Duration
	}{
		{"a 429 defers the listing page", "limited", time.Minute},
		{"a blocked gate defers the listing page", "linkedin", 10 * time.Minute},
	}
	for _, tt := range listings {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, http.StatusOK, "")
			target := f.target(t, tt.source)
			task := queuetest.ListingTask(tt.source)
			task.TargetID, task.RunID = target.ID, target.RunID
			start := time.Now()
			err := f.processor.Process(t.Context(), task)
			var deferred *queue.DeferError
			if !errors.As(err, &deferred) {
				t.Fatalf("Process() = %v, want a deferral", err)
			}
			if wait := deferred.Until.Sub(start); wait < tt.want-5*time.Second || wait > tt.want+5*time.Second {
				t.Errorf("deferred for %v, want about %v", wait, tt.want)
			}
			got := f.runStatus(t, target)
			if got.RunStatus != "running" || !strings.HasPrefix(got.LastRunError, "waiting until ") {
				t.Errorf("run = %q %q, want running and waiting until …", got.RunStatus, got.LastRunError)
			}
		})
	}

	t.Run("a blocked gate defers the detail task", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t, http.StatusOK, "")
		err := f.processor.Process(t.Context(), queuetest.DetailTask("blocked"))
		var deferred *queue.DeferError
		if !errors.As(err, &deferred) {
			t.Fatalf("Process() = %v, want a deferral", err)
		}
		if got := f.ingest.requests.Load(); got != 0 {
			t.Errorf("ingest requests = %d, want 0", got)
		}
	})

	t.Run("a deferred first page resumes and clears the wait", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		target := f.target(t, "linkedin")
		task := queuetest.ListingTask("linkedin")
		task.TargetID, task.RunID = target.ID, target.RunID
		var deferred *queue.DeferError
		if err := f.processor.Process(t.Context(), task); !errors.As(err, &deferred) {
			t.Fatalf("first Process() = %v, want a deferral", err)
		}
		if err := f.processor.Process(t.Context(), task); err != nil {
			t.Fatalf("second Process() = %v, want nil", err)
		}
		if got := f.runStatus(t, target); got.RunStatus != "succeeded" || got.LastRunError != "" {
			t.Errorf("run = %q %q, want succeeded with no message", got.RunStatus, got.LastRunError)
		}
	})

	t.Run("an unparseable board is acked, logged and fails its run", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t, http.StatusOK, "")
		exporter := scraper.NewAPIExporter(f.ingest.server.URL, "token").WithInitialBackoff(0)
		unparseable := fmt.Errorf("wttj: no Company in state: %w", sources.ErrUnparseable)
		var logs bytes.Buffer
		p := worker.NewProcessor(worker.Deps{
			JS:     jobsearch.Build(jobsearchtest.NewDeps(f.store)),
			Boards: scraper.NewBoardPoller(f.store, failingBoard{err: unparseable}, exporter),
			Logger: slog.New(slog.NewTextHandler(&logs, nil)),
		})
		target := f.target(t, "greenhouse")
		task := queue.Task{Version: 1, Source: "greenhouse", Kind: queue.BoardCheckTask, BoardID: f.board(t), Manual: true, TargetID: target.ID, RunID: target.RunID}
		if err := p.Process(t.Context(), task); err != nil {
			t.Fatalf("Process() = %v, want nil so the task is acked", err)
		}
		if got := f.runStatus(t, target).RunStatus; got != "failed" {
			t.Errorf("run status = %q, want failed", got)
		}
		if !strings.Contains(logs.String(), "board page unparseable") {
			t.Errorf("logs = %q, want a board page unparseable warning", logs.String())
		}
	})

	boards := []struct {
		name      string
		err       error
		wantDefer bool
	}{
		{"a 429 defers the board check", limited, true},
		{"any other failure retries the board check", errors.New("boom"), false},
	}
	for _, tt := range boards {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, http.StatusOK, "")
			exporter := scraper.NewAPIExporter(f.ingest.server.URL, "token").WithInitialBackoff(0)
			p := worker.NewProcessor(worker.Deps{
				JS:     jobsearch.Build(jobsearchtest.NewDeps(f.store)),
				Boards: scraper.NewBoardPoller(f.store, failingBoard{err: tt.err}, exporter),
			})
			err := p.Process(ctx, queue.Task{Version: 1, Source: "greenhouse", Kind: queue.BoardCheckTask, BoardID: f.board(t), Manual: true})
			var deferred *queue.DeferError
			if err == nil || errors.As(err, &deferred) != tt.wantDefer {
				t.Errorf("Process() = %v, want deferral %v and an error", err, tt.wantDefer)
			}
		})
	}
}

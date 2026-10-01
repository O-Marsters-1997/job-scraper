package worker_test

import (
	"context"
	"errors"
	"fmt"
	"io"
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

func (d detailStub) GetDetails(_ context.Context, url string) (dto.Job, error) {
	return dto.Job{Source: "wis", Title: "Job", URL: url}, d.err
}

const fetchedURL = "https://8.8.8.8/job"

type fetchingDetailer struct{ client *http.Client }

func newFetchingDetailer(cache proxy.Cache) fetchingDetailer {
	upstream := identitytest.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("page"))}, nil
	})
	return fetchingDetailer{client: &http.Client{Transport: proxy.NewFetchTransport(upstream, nil, cache)}}
}

func (d fetchingDetailer) GetDetails(ctx context.Context, url string) (dto.Job, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchedURL, nil)
	if err != nil {
		return dto.Job{}, err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return dto.Job{}, err
	}
	_ = resp.Body.Close()
	return dto.Job{Source: "wis", Title: "Job", URL: url}, nil
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

func (oneBoardJob) FetchBoard(context.Context, dto.BoardPoll) ([]dto.Job, time.Duration, error) {
	return []dto.Job{{Title: "Engineer", URL: "https://example.com/1"}}, 0, nil
}

type ingest struct {
	server   *httptest.Server
	requests atomic.Int32
}

func newIngest(t *testing.T, status int) *ingest {
	t.Helper()
	in := &ingest{}
	in.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		in.requests.Add(1)
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
	build := func(target dto.SourceTarget) (sources.Source, bool) {
		switch target.Source {
		case "fetching":
			return page, true
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
		CardComplete: map[string]bool{"remoteok": true},
		Detailers: map[string]sources.DetailFetcher{
			"wis":      detailStub{},
			"indeed":   newFetchingDetailer(store),
			"linkedin": detailStub{err: errors.New("page gone")},
			"gone":     detailStub{err: fmt.Errorf("%w: status 404", sources.ErrGone)},
		},
	})
	return fixture{store: store, processor: processor, ingest: in, published: published}
}

func (f fixture) discoverProcessor(discover worker.DiscoverFunc, scoring worker.IncludeFilterConfigs) *worker.Processor {
	return worker.NewProcessor(worker.Deps{JS: jobsearch.Build(jobsearchtest.NewDeps(f.store)), Scoring: scoring, Discover: discover})
}

func (f fixture) target(t *testing.T, source string) dto.SourceTarget {
	t.Helper()
	target, err := f.store.CreateSourceTargetWithRun(t.Context(), "user-1", source, "https://example.com/search", true, nil)
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
	ctx := context.Background()

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
}

type configsStub []dto.SearchConfig

func (c configsStub) IncludeFilterConfigs(context.Context) ([]dto.SearchConfig, error) { return c, nil }

func TestProcessBoardDiscover(t *testing.T) {
	ctx := context.Background()
	jobs := []dto.Job{{Title: "Go Engineer", Location: "London"}}
	discover := func(context.Context, string, string) (string, []dto.Job, error) {
		return "Acme Corp", jobs, nil
	}
	task := queue.Task{Version: 1, Source: "greenhouse", Kind: queue.BoardDiscoverTask, BoardToken: "acme"}
	match := dto.SearchConfig{UserID: "match", RequiredTitleKeywords: []string{"go"}}
	miss := dto.SearchConfig{UserID: "miss", RequiredTitleKeywords: []string{"rust"}}
	prior := dto.SearchConfig{UserID: "prior", RequiredTitleKeywords: []string{"go"}}

	t.Run("tracks matching users as new and leaves existing rows alone", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		company, err := f.store.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme-corp", Name: "Acme Corp"})
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

	t.Run("a board matching nobody stays verified and untracked", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		p := f.discoverProcessor(discover, configsStub{miss})
		if err := p.Process(ctx, task); err != nil {
			t.Fatalf("Process() = %v, want nil", err)
		}
		tracked, err := f.store.ListTrackedCompaniesForUser(ctx, "miss")
		if err != nil || len(tracked) != 0 {
			t.Fatalf("ListTrackedCompaniesForUser(miss) = %+v, %v, want none", tracked, err)
		}
	})

	t.Run("a failed fetch is acked without creating a company", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		failing := func(context.Context, string, string) (string, []dto.Job, error) {
			return "", nil, errors.New("board gone")
		}
		if err := f.discoverProcessor(failing, configsStub{match}).Process(ctx, task); err != nil {
			t.Fatalf("Process() = %v, want nil", err)
		}
		companies, err := f.store.ListCompaniesForUser(ctx, "match")
		if err != nil || len(companies) != 0 {
			t.Fatalf("ListCompaniesForUser() = %+v, %v, want none", companies, err)
		}
	})
}

func TestFailRun(t *testing.T) {
	ctx := context.Background()

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

	t.Run("run already gone is not an error", func(t *testing.T) {
		f := newFixture(t, http.StatusOK, "")
		task := queue.Task{Source: "wis", Kind: queue.ListingPageTask, TargetID: "gone", RunID: "gone"}
		if err := f.processor.FailRun(ctx, task); err != nil {
			t.Fatal(err)
		}
	})
}

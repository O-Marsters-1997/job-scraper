package scraper_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
	"github.com/ollymarsters/job-scraper/internal/worker/scraper"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

type boardStoreStub struct {
	board    dto.BoardPoll
	state    string
	snapshot dto.BoardSnapshot
}

func (s *boardStoreStub) ClaimBoard(context.Context, string, bool) (dto.BoardPoll, error) {
	return s.board, nil
}
func (s *boardStoreStub) ClearBoardCheckPublished(context.Context, string) error {
	return nil
}
func (s *boardStoreStub) CompleteBoard(_ context.Context, snapshot dto.BoardSnapshot) error {
	s.snapshot = snapshot
	s.state = "completed"
	return nil
}
func (s *boardStoreStub) FailBoard(context.Context, dto.BoardPoll) error {
	s.state = "failed"
	return nil
}

type boardFetcherStub struct {
	jobs     []dto.Job
	hint     time.Duration
	reported int
	err      error
}

func (f boardFetcherStub) FetchBoard(context.Context, dto.BoardPoll) (sources.BoardResult, error) {
	return sources.BoardResult{Jobs: f.jobs, NextPollIn: f.hint, Reported: f.reported}, f.err
}

type boardIngesterStub struct {
	err  error
	jobs []dto.Job
}

func (i *boardIngesterStub) BulkExport(_ context.Context, jobs []dto.Job) error {
	i.jobs = jobs
	return i.err
}

var (
	errPartial = errors.New("partial page")
	errPersist = errors.New("persistence failed")
)

func TestBoardPollCompletesOnlyAfterFetchAndIngest(t *testing.T) {
	for _, tc := range []struct {
		name      string
		fetchErr  error
		ingestErr error
		want      string
		wantErr   error
	}{
		{"fetch failed", errPartial, nil, "failed", errPartial},
		{"ingest failed", nil, errPersist, "failed", errPersist},
		{"complete", nil, nil, "completed", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &boardStoreStub{board: dto.BoardPoll{ID: "board", CompanyID: "company", Source: "greenhouse", Token: "acme"}}
			poller := scraper.NewBoardPoller(store, boardFetcherStub{jobs: []dto.Job{{Title: "Engineer", URL: "https://example.com/1"}}, err: tc.fetchErr}, &boardIngesterStub{err: tc.ingestErr})
			err := poller.PollBoard(t.Context(), "board", false)
			if tc.wantErr == nil && err != nil || tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Errorf("PollBoard() = %v, want %v", err, tc.wantErr)
			}
			if store.state != tc.want {
				t.Fatalf("board state = %q, want %q", store.state, tc.want)
			}
		})
	}
}

func TestBoardPollUnclaimableCheck(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		manual  bool
		wantErr error
		wantDue int
	}{
		{"scheduled check is acked and makes the board due again", false, nil, 1},
		{"manual check errors and leaves the board held", true, jobsearch.ErrBoardClaimUnavailable, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			st := jobsearchtest.NewFakeStore()
			company, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme"})
			if err != nil {
				t.Fatal(err)
			}
			board, err := st.UpsertCandidateBoard(ctx, company.ID, "greenhouse", "acme")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := st.VerifyCompanyBoard(ctx, company.ID, "greenhouse", "acme", "test", ""); err != nil {
				t.Fatal(err)
			}
			if err := st.MarkBoardCheckPublished(ctx, board.ID); err != nil {
				t.Fatal(err)
			}
			held, err := st.ClaimBoard(ctx, board.ID, true)
			if err != nil {
				t.Fatal(err)
			}

			poller := scraper.NewBoardPoller(st, boardFetcherStub{}, &boardIngesterStub{})
			if err := poller.PollBoard(ctx, board.ID, tc.manual); !errors.Is(err, tc.wantErr) {
				t.Errorf("PollBoard() = %v, want %v", err, tc.wantErr)
			}
			if err := st.CompleteBoard(ctx, dto.BoardSnapshot{Poll: held, Complete: true}); err != nil {
				t.Fatal(err)
			}
			if due, err := st.ListDueBoards(ctx); err != nil || len(due) != tc.wantDue {
				t.Errorf("ListDueBoards() = %+v, %v, want %d due", due, err, tc.wantDue)
			}
		})
	}
}

func TestBoardPollCarriesVerifiedCompanyIdentity(t *testing.T) {
	store := &boardStoreStub{board: dto.BoardPoll{ID: "board", CompanyID: "company", CompanySlug: "company-slug", Source: "greenhouse", Token: "regional-token"}}
	ingester := &boardIngesterStub{}
	poller := scraper.NewBoardPoller(store, boardFetcherStub{jobs: []dto.Job{{Title: "Engineer", URL: "https://example.com/1", CompanySlug: "regional-token"}}}, ingester)
	if err := poller.PollBoard(t.Context(), "board", false); err != nil {
		t.Fatal(err)
	}
	if len(ingester.jobs) != 1 || ingester.jobs[0].CompanyID != "company" || ingester.jobs[0].CompanySlug != "company-slug" || ingester.jobs[0].BoardID != "board" {
		t.Fatalf("ingested job identity=%v", ingester.jobs)
	}
}

func TestBoardPollPassesNextPollHintOnSuccessOnly(t *testing.T) {
	hint := 72 * time.Hour
	store := &boardStoreStub{board: dto.BoardPoll{ID: "board"}}
	poller := scraper.NewBoardPoller(store, boardFetcherStub{hint: hint}, &boardIngesterStub{})
	if err := poller.PollBoard(t.Context(), "board", false); err != nil {
		t.Fatal(err)
	}
	if store.snapshot.NextPollIn != hint {
		t.Errorf("snapshot NextPollIn = %v, want %v", store.snapshot.NextPollIn, hint)
	}

	failed := &boardStoreStub{board: dto.BoardPoll{ID: "board"}}
	poller = scraper.NewBoardPoller(failed, boardFetcherStub{hint: hint, err: errPartial}, &boardIngesterStub{})
	_ = poller.PollBoard(t.Context(), "board", false)
	if failed.state != "failed" || failed.snapshot.NextPollIn != 0 {
		t.Errorf("failed poll state = %q, hint = %v, want failed and no hint", failed.state, failed.snapshot.NextPollIn)
	}
}

func TestPollPrefetchedCompletesTheFullListWithHint(t *testing.T) {
	hint := 72 * time.Hour
	store := &boardStoreStub{board: dto.BoardPoll{ID: "board", CompanyID: "company", CompanySlug: "company-slug"}}
	ingester := &boardIngesterStub{}
	poller := scraper.NewBoardPoller(store, boardFetcherStub{err: errPartial}, ingester)
	jobs := []dto.Job{{Title: "Engineer", URL: "https://example.com/1"}, {Title: "Designer", URL: "https://example.com/2"}}
	if err := poller.PollPrefetched(t.Context(), "board", jobs, hint); err != nil {
		t.Fatal(err)
	}
	if len(ingester.jobs) != 2 || ingester.jobs[0].CompanyID != "company" || ingester.jobs[0].BoardID != "board" {
		t.Errorf("ingested jobs = %+v, want both stamped with the board's identity", ingester.jobs)
	}
	if store.state != "completed" || !store.snapshot.Complete || store.snapshot.NextPollIn != hint {
		t.Errorf("state = %q, snapshot = %+v, want a complete snapshot with hint %v", store.state, store.snapshot, hint)
	}
}

func TestPollPrefetchedStampsTheJobMatchingTheListing(t *testing.T) {
	t.Parallel()
	listing := scraper.Listing{Via: "wttj", URL: "https://wttj.example/jobs/1", ApplyURL: "https://boards.greenhouse.io/acme/jobs/123"}
	tests := []struct {
		name string
		job  dto.Job
		want bool
	}{
		{"same apply url", dto.Job{URL: listing.ApplyURL}, true},
		{"same posting id", dto.Job{URL: "https://job-boards.greenhouse.io/acme/jobs/123?gh_src=x", ProviderPostingID: "123"}, true},
		{"other posting", dto.Job{URL: "https://boards.greenhouse.io/acme/jobs/999", ProviderPostingID: "999"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := &boardStoreStub{board: dto.BoardPoll{ID: "board"}}
			ingester := &boardIngesterStub{}
			poller := scraper.NewBoardPoller(store, boardFetcherStub{}, ingester)
			if err := poller.PollPrefetched(t.Context(), "board", []dto.Job{tt.job}, 0, listing); err != nil {
				t.Fatal(err)
			}
			got := ingester.jobs[0]
			if stamped := got.DiscoveredVia == "wttj" && got.DiscoveryURL == listing.URL; stamped != tt.want {
				t.Errorf("job = %+v, stamped = %v, want %v", got, stamped, tt.want)
			}
		})
	}
}

func TestBoardPollLogsUnderparsedOnlyWhenShort(t *testing.T) {
	jobs := func(n int) []dto.Job {
		out := make([]dto.Job, n)
		for i := range out {
			out[i] = dto.Job{Title: "Engineer", URL: fmt.Sprintf("https://example.com/%d", i)}
		}
		return out
	}
	for _, tc := range []struct {
		name      string
		parsed    int
		reported  int
		wantEvent bool
	}{
		{"short poll", 90, 100, true},
		{"full poll", 100, 100, false},
		{"within tolerance", 99, 100, false},
		{"unknown reported", 3, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			prev := slog.Default()
			slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
			t.Cleanup(func() { slog.SetDefault(prev) })

			store := &boardStoreStub{board: dto.BoardPoll{ID: "board"}}
			poller := scraper.NewBoardPoller(store, boardFetcherStub{jobs: jobs(tc.parsed), reported: tc.reported}, &boardIngesterStub{})
			if err := poller.PollBoard(t.Context(), "board", false); err != nil {
				t.Fatal(err)
			}

			var logged bool
			for line := range bytes.SplitSeq(buf.Bytes(), []byte("\n")) {
				var rec map[string]any
				if json.Unmarshal(line, &rec) == nil && rec["event"] == telemetry.EventBoardUnderparsed {
					logged = true
				}
			}
			if logged != tc.wantEvent {
				t.Errorf("board.underparsed logged = %t, want %t", logged, tc.wantEvent)
			}
			if store.snapshot.Reported != tc.reported {
				t.Errorf("snapshot Reported = %d, want %d", store.snapshot.Reported, tc.reported)
			}
		})
	}
}

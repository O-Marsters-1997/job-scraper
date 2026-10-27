package jobsearch_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
)

type scrapeRunsStore struct {
	*jobsearchtest.FakeStore
	runs []dto.ScrapeRun
	got  dto.ScrapeRunListOptions

	summaryGot dto.ScrapeRunSummaryOptions
}

func (s *scrapeRunsStore) ScrapeRunSummary(_ context.Context, _ string, opts dto.ScrapeRunSummaryOptions) (dto.ScrapeRunSummary, error) {
	s.summaryGot = opts
	return dto.ScrapeRunSummary{}, nil
}

func (s *scrapeRunsStore) ListScrapeRuns(_ context.Context, _ string, opts dto.ScrapeRunListOptions) ([]dto.ScrapeRun, error) {
	s.got = opts
	return s.runs, nil
}

func TestService_ListScrapeRuns(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 1, 2, 3, 4, 5, 6000, time.UTC)
	runs := []dto.ScrapeRun{{ID: "a", StartedAt: start}, {ID: "b", StartedAt: start.Add(-time.Hour)}, {ID: "c", StartedAt: start.Add(-2 * time.Hour)}}

	t.Run("a full page hands back a cursor that resumes after its last item", func(t *testing.T) {
		t.Parallel()
		st := &scrapeRunsStore{FakeStore: jobsearchtest.NewFakeStore(), runs: runs}
		svc := jobsearch.NewService(st, nil, jobsearchtest.NewNoopScoring())
		page, err := svc.ListScrapeRuns(t.Context(), userID, dto.ScrapeRunsQuery{Limit: "2"})
		if err != nil || len(page.Items) != 2 || page.NextCursor == nil {
			t.Fatalf("ListScrapeRuns() = %+v, %v, want 2 items and a cursor", page, err)
		}
		if st.got.Limit != 3 {
			t.Errorf("store limit = %d, want 3 (one extra to detect another page)", st.got.Limit)
		}

		if _, err := svc.ListScrapeRuns(t.Context(), userID, dto.ScrapeRunsQuery{Before: *page.NextCursor}); err != nil {
			t.Fatal(err)
		}
		if st.got.BeforeID != "b" || !st.got.BeforeStartedAt.Equal(runs[1].StartedAt) {
			t.Errorf("resumed after (%v, %q), want (%v, b)", st.got.BeforeStartedAt, st.got.BeforeID, runs[1].StartedAt)
		}
	})

	t.Run("a short page has no cursor", func(t *testing.T) {
		t.Parallel()
		svc := jobsearch.NewService(&scrapeRunsStore{FakeStore: jobsearchtest.NewFakeStore(), runs: runs[:1]}, nil, jobsearchtest.NewNoopScoring())
		page, err := svc.ListScrapeRuns(t.Context(), userID, dto.ScrapeRunsQuery{})
		if err != nil || len(page.Items) != 1 || page.NextCursor != nil {
			t.Errorf("ListScrapeRuns() = %+v, %v, want 1 item and no cursor", page, err)
		}
	})

	t.Run("rejects a bad limit or cursor", func(t *testing.T) {
		t.Parallel()
		svc := jobsearch.NewService(jobsearchtest.NewFakeStore(), nil, jobsearchtest.NewNoopScoring())
		for _, q := range []dto.ScrapeRunsQuery{{Limit: "0"}, {Before: "!!"}, {Before: "bm90LWEtY3Vyc29y"}} {
			_, err := svc.ListScrapeRuns(t.Context(), userID, q)
			if !apperr.IsKind(err, apperr.KindInvalid) {
				t.Errorf("ListScrapeRuns(%+v) err = %v, want invalid", q, err)
			}
		}
	})
}

type healthStore struct {
	*jobsearchtest.FakeStore
	rows  []dto.SourceHealth
	fresh []dto.SourceNewJob
}

func (s *healthStore) ListRecentNewJobs(context.Context, string, time.Time) ([]dto.SourceNewJob, error) {
	return s.fresh, nil
}

func TestService_SourceHealth_New24h(t *testing.T) {
	t.Parallel()
	scoring := jobsearchtest.NewNoopScoring()
	scoring.SeedSearchConfig(dto.SearchConfig{UserID: userID, ExcludedTitleKeywords: []string{"manager"}})
	st := &healthStore{
		FakeStore: jobsearchtest.NewFakeStore(),
		rows:      []dto.SourceHealth{{TargetID: "t1"}, {Kind: dto.ScrapeRunBoard, BoardID: "b1"}, {TargetID: "t2"}},
		fresh: []dto.SourceNewJob{
			{TargetID: "t1", Job: dto.Job{Title: "Go Engineer"}},
			{TargetID: "t1", Job: dto.Job{Title: "Engineering Manager"}},
			{BoardID: "b1", Job: dto.Job{Title: "Go Engineer"}},
			{BoardID: "b1", Job: dto.Job{Title: "Rust Engineer"}},
		},
	}
	got, err := jobsearch.NewService(st, nil, scoring).SourceHealth(t.Context(), userID)
	if err != nil || len(got) != 3 {
		t.Fatalf("SourceHealth() = %v, %v, want three rows", got, err)
	}
	for i, want := range []int{1, 2, 0} {
		if got[i].New24h != want {
			t.Errorf("SourceHealth()[%d].New24h = %d, want %d", i, got[i].New24h, want)
		}
	}
}

func (s *healthStore) ListSourceHealth(context.Context, string) ([]dto.SourceHealth, error) {
	return s.rows, nil
}

func TestService_SourceHealth(t *testing.T) {
	t.Parallel()
	ok, retried, failed := dto.OutcomeSucceeded, dto.OutcomeRetried, dto.OutcomeFailed
	tests := []struct {
		name    string
		recent  []dto.ScrapeRunOutcome
		new     int
		retries int
		status  dto.HealthStatus
		reason  dto.HealthReason
	}{
		{"no finished runs is green", nil, 0, 0, dto.HealthGreen, dto.ReasonNoRuns},
		{"last run failed", []dto.ScrapeRunOutcome{ok, ok, failed}, 4, 3, dto.HealthRed, dto.ReasonLastFailed},
		{"two failures in a row", []dto.ScrapeRunOutcome{ok, failed, failed, ok}, 4, 6, dto.HealthRed, dto.ReasonFailingStreak},
		{"separated failures are not a streak", []dto.ScrapeRunOutcome{failed, ok, failed, ok}, 4, 6, dto.HealthAmber, dto.ReasonRetried},
		{"a retried attempt", []dto.ScrapeRunOutcome{ok, retried, ok}, 4, 1, dto.HealthAmber, dto.ReasonRetried},
		{"five runs with nothing new", []dto.ScrapeRunOutcome{ok, ok, ok, ok, ok}, 0, 0, dto.HealthAmber, dto.ReasonNoNew},
		{"fewer than five runs with nothing new is healthy", []dto.ScrapeRunOutcome{ok, ok, ok, ok}, 0, 0, dto.HealthGreen, dto.ReasonOK},
		{"five runs with one new", []dto.ScrapeRunOutcome{ok, ok, ok, ok, ok}, 1, 0, dto.HealthGreen, dto.ReasonOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			st := &healthStore{FakeStore: jobsearchtest.NewFakeStore(), rows: []dto.SourceHealth{{Recent: tt.recent, New: tt.new, Retries: tt.retries}}}
			svc := jobsearch.NewService(st, nil, jobsearchtest.NewNoopScoring())
			got, err := svc.SourceHealth(t.Context(), userID)
			if err != nil || len(got) != 1 {
				t.Fatalf("SourceHealth() = %v, %v, want one row", got, err)
			}
			if got[0].Status != tt.status || got[0].Reason != tt.reason {
				t.Errorf("SourceHealth() = %s/%s, want %s/%s", got[0].Status, got[0].Reason, tt.status, tt.reason)
			}
		})
	}

	t.Run("transient failures", func(t *testing.T) {
		t.Parallel()
		f, ok := dto.OutcomeFailed, dto.OutcomeSucceeded
		for _, tt := range []struct {
			name      string
			recent    []dto.ScrapeRunOutcome
			transient []bool
			status    dto.HealthStatus
			reason    dto.HealthReason
		}{
			{"two transient failures are amber", []dto.ScrapeRunOutcome{ok, f, f}, []bool{false, true, true}, dto.HealthAmber, dto.ReasonTransient},
			{"three transient failures in a row are red", []dto.ScrapeRunOutcome{ok, f, f, f}, []bool{false, true, true, true}, dto.HealthRed, dto.ReasonFailingStreak},
			{"a hard failure last is red", []dto.ScrapeRunOutcome{f, f}, []bool{true, false}, dto.HealthRed, dto.ReasonLastFailed},
			{"a transient failure followed by success is healthy", []dto.ScrapeRunOutcome{ok, f, ok}, []bool{false, true, false}, dto.HealthGreen, dto.ReasonOK},
			{"a hard pair is red even when not last", []dto.ScrapeRunOutcome{f, f, ok}, []bool{false, false, false}, dto.HealthRed, dto.ReasonFailingStreak},
			{"a transient failure between hard ones breaks the pair", []dto.ScrapeRunOutcome{f, f, ok}, []bool{false, true, false}, dto.HealthGreen, dto.ReasonOK},
		} {
			st := &healthStore{FakeStore: jobsearchtest.NewFakeStore(), rows: []dto.SourceHealth{{Recent: tt.recent, Transient: tt.transient, New: 1}}}
			got, err := jobsearch.NewService(st, nil, jobsearchtest.NewNoopScoring()).SourceHealth(t.Context(), userID)
			if err != nil || len(got) != 1 {
				t.Fatalf("%s: SourceHealth() = %v, %v", tt.name, got, err)
			}
			if got[0].Status != tt.status || got[0].Reason != tt.reason {
				t.Errorf("%s: SourceHealth() = %s/%s, want %s/%s", tt.name, got[0].Status, got[0].Reason, tt.status, tt.reason)
			}
		}
	})

	t.Run("no searches is an empty list", func(t *testing.T) {
		t.Parallel()
		svc := jobsearch.NewService(jobsearchtest.NewFakeStore(), nil, jobsearchtest.NewNoopScoring())
		got, err := svc.SourceHealth(t.Context(), userID)
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("SourceHealth() = %#v, %v, want empty non-nil", got, err)
		}
	})
}

func TestService_ScrapeRunSummary(t *testing.T) {
	t.Parallel()
	svc := func(st *scrapeRunsStore) *jobsearch.Service {
		return jobsearch.NewService(st, nil, jobsearchtest.NewNoopScoring())
	}

	t.Run("bucket is the smallest that keeps the range to 48 buckets", func(t *testing.T) {
		t.Parallel()
		for rng, want := range map[string]time.Duration{
			"":    6 * time.Hour,
			"1h":  5 * time.Minute,
			"6h":  15 * time.Minute,
			"24h": time.Hour,
			"7d":  6 * time.Hour,
			"30d": 24 * time.Hour,
		} {
			st := &scrapeRunsStore{FakeStore: jobsearchtest.NewFakeStore()}
			if _, err := svc(st).ScrapeRunSummary(t.Context(), userID, dto.ScrapeRunsQuery{Range: rng}); err != nil {
				t.Fatalf("range %q: %v", rng, err)
			}
			if st.summaryGot.Bucket != want {
				t.Errorf("range %q bucket = %v, want %v", rng, st.summaryGot.Bucket, want)
			}
		}
	})

	t.Run("custom range replaces the preset and facets split on commas", func(t *testing.T) {
		t.Parallel()
		st := &scrapeRunsStore{FakeStore: jobsearchtest.NewFakeStore()}
		q := dto.ScrapeRunsQuery{
			From: "2026-01-01T00:00:00Z", To: "2026-01-02T00:00:00Z", Range: "1h",
			Outcome: "failed,retried", Source: "linkedin",
		}
		if _, err := svc(st).ScrapeRunSummary(t.Context(), userID, q); err != nil {
			t.Fatal(err)
		}
		got := st.summaryGot
		if !got.From.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) || !got.To.Equal(got.End) || got.Bucket != time.Hour {
			t.Errorf("options = %+v, want 1 Jan to 2 Jan in 1h buckets", got)
		}
		if !slices.Equal(got.Outcome, []string{"failed", "retried"}) || !slices.Equal(got.Source, []string{"linkedin"}) {
			t.Errorf("facets = %v, %v", got.Outcome, got.Source)
		}
	})

	t.Run("rejects a bad range, facet value or custom range", func(t *testing.T) {
		t.Parallel()
		for _, q := range []dto.ScrapeRunsQuery{
			{Range: "2d"},
			{From: "2026-01-01T00:00:00Z", To: "2026-03-01T00:00:00Z"},
			{From: "2026-01-02T00:00:00Z", To: "2026-01-01T00:00:00Z"},
			{From: "2026-01-01T00:00:00Z"},
			{Outcome: "failed,exploded"},
			{Kind: "crawl"},
			{Source: "nowhere"},
			{Trigger: "cron"},
		} {
			if _, err := svc(&scrapeRunsStore{FakeStore: jobsearchtest.NewFakeStore()}).ScrapeRunSummary(t.Context(), userID, q); !apperr.IsKind(err, apperr.KindInvalid) {
				t.Errorf("ScrapeRunSummary(%+v) err = %v, want invalid", q, err)
			}
			if _, err := svc(&scrapeRunsStore{FakeStore: jobsearchtest.NewFakeStore()}).ListScrapeRuns(t.Context(), userID, q); !apperr.IsKind(err, apperr.KindInvalid) {
				t.Errorf("ListScrapeRuns(%+v) err = %v, want invalid", q, err)
			}
		}
	})
}

type snapshotsStore struct {
	*jobsearchtest.FakeStore
	err error
}

func (s snapshotsStore) ListScrapeRunJobSnapshots(context.Context, string, string) ([]dto.ScrapeRunJobSnapshot, error) {
	return nil, s.err
}

func TestService_ListScrapeRunJobSnapshots(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		kind apperr.Kind
	}{
		{"an untracked run is not found", data.ErrNotFound, apperr.KindNotFound},
		{"a malformed id is invalid", store.ErrInvalidID, apperr.KindInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := jobsearch.NewService(snapshotsStore{FakeStore: jobsearchtest.NewFakeStore(), err: tt.err}, nil, jobsearchtest.NewNoopScoring())
			if _, err := svc.ListScrapeRunJobSnapshots(t.Context(), userID, "run"); !apperr.IsKind(err, tt.kind) {
				t.Errorf("ListScrapeRunJobSnapshots() err = %v, want kind %v", err, tt.kind)
			}
		})
	}

	t.Run("no changes is an empty list, not null", func(t *testing.T) {
		t.Parallel()
		svc := jobsearch.NewService(snapshotsStore{FakeStore: jobsearchtest.NewFakeStore()}, nil, jobsearchtest.NewNoopScoring())
		got, err := svc.ListScrapeRunJobSnapshots(t.Context(), userID, "run")
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("ListScrapeRunJobSnapshots() = %#v, %v, want an empty non-nil list", got, err)
		}
	})
}

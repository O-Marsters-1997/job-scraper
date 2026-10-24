package jobsearch_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
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

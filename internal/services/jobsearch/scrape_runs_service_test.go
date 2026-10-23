package jobsearch_test

import (
	"context"
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

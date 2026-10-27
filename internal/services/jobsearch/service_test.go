package jobsearch_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
)

func seedJobs(t *testing.T, n int) *jobsearchtest.FakeStore {
	t.Helper()
	st := jobsearchtest.NewFakeStore()
	for i := range n {
		job := dto.Job{
			Title: "Role", URL: "https://example.com/" + string(rune('a'+i)),
			ScrapedAt: time.Date(2026, 1, i+1, 0, 0, 0, 0, time.UTC),
		}
		if _, _, err := st.SaveCanonical(t.Context(), job); err != nil {
			t.Fatalf("SaveCanonical(%s) err = %v", job.URL, err)
		}
	}
	return st
}

func TestList(t *testing.T) {
	t.Run("rejects bad pagination", func(t *testing.T) {
		tests := []struct {
			name  string
			query dto.JobsQuery
		}{
			{"page zero", dto.JobsQuery{Page: "0"}},
			{"page not a number", dto.JobsQuery{Page: "next"}},
			{"page offset overflows", dto.JobsQuery{Page: "2147483647"}},
			{"page_size zero", dto.JobsQuery{PageSize: "0"}},
			{"unknown graded", dto.JobsQuery{Graded: "maybe"}},
			{"unknown seen", dto.JobsQuery{Seen: "maybe"}},
			{"unknown poster", dto.JobsQuery{Poster: "recruiter"}},
			{"suit not a number", dto.JobsQuery{Suit: "high"}},
			{"salary_min not a number", dto.JobsQuery{SalaryMin: "lots"}},
		}
		svc := jobsearch.NewService(jobsearchtest.NewFakeStore(), nil, jobsearchtest.NewNoopScoring())
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := svc.List(t.Context(), userID, tt.query)
				if !apperr.IsKind(err, apperr.KindInvalid) {
					t.Errorf("List(%+v) err = %v, want kind %v", tt.query, err, apperr.KindInvalid)
				}
			})
		}
	})

	t.Run("pages with offset and total", func(t *testing.T) {
		t.Parallel()
		svc := jobsearch.NewService(seedJobs(t, 3), nil, jobsearchtest.NewNoopScoring())

		for _, tt := range []struct {
			name      string
			query     dto.JobsQuery
			wantItems int
		}{
			{"first page", dto.JobsQuery{Page: "1", PageSize: "2"}, 2},
			{"last page", dto.JobsQuery{Page: "2", PageSize: "2"}, 1},
			{"past the end", dto.JobsQuery{Page: "9", PageSize: "2"}, 0},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				page, err := svc.List(t.Context(), userID, tt.query)
				if err != nil {
					t.Fatalf("List(%+v) err = %v", tt.query, err)
				}
				if len(page.Items) != tt.wantItems || page.Total == nil || *page.Total != 3 {
					t.Errorf("List(%+v) = %d items, total %v, want %d items, total 3", tt.query, len(page.Items), page.Total, tt.wantItems)
				}
			})
		}
	})

	t.Run("skip_total=1 leaves the total null", func(t *testing.T) {
		t.Parallel()
		svc := jobsearch.NewService(seedJobs(t, 3), nil, jobsearchtest.NewNoopScoring())
		page, err := svc.List(t.Context(), userID, dto.JobsQuery{Page: "2", PageSize: "2", SkipTotal: "1"})
		if err != nil {
			t.Fatalf("List(skip_total=1) err = %v", err)
		}
		if len(page.Items) != 1 || page.Total != nil {
			t.Errorf("List(skip_total=1) = %d items, total %v, want 1 item, nil total", len(page.Items), page.Total)
		}
	})
}

func TestTriageQueue(t *testing.T) {
	t.Parallel()

	svc := jobsearch.NewService(jobsearchtest.NewFakeStore(), nil, jobsearchtest.NewNoopScoring())

	t.Run("accepts the poster values", func(t *testing.T) {
		t.Parallel()
		for _, poster := range []string{"", "direct", "agency"} {
			if _, err := svc.TriageQueue(t.Context(), userID, dto.TriageQuery{Poster: poster}); err != nil {
				t.Errorf("TriageQueue(poster=%q) err = %v", poster, err)
			}
		}
	})

	t.Run("rejects any other poster", func(t *testing.T) {
		t.Parallel()
		for _, poster := range []string{"all", "recruiter"} {
			_, err := svc.TriageQueue(t.Context(), userID, dto.TriageQuery{Poster: poster})
			if !apperr.IsKind(err, apperr.KindInvalid) {
				t.Errorf("TriageQueue(poster=%q) err = %v, want kind %v", poster, err, apperr.KindInvalid)
			}
		}
	})
}

func TestGet(t *testing.T) {
	t.Run("missing job is not found", func(t *testing.T) {
		svc := jobsearch.NewService(jobsearchtest.NewFakeStore(), nil, jobsearchtest.NewNoopScoring())
		_, err := svc.Get(t.Context(), userID, "missing")
		if !apperr.IsKind(err, apperr.KindNotFound) {
			t.Fatalf("Get(missing) err = %v, want kind %v", err, apperr.KindNotFound)
		}
	})

	t.Run("returns every listing in first-seen order", func(t *testing.T) {
		st := jobsearchtest.NewFakeStore()
		posting := dto.Job{Title: "Engineer", BoardID: "board-1", ProviderPostingID: "p-1"}
		var saved dto.Job
		for _, listing := range []struct{ url, source string }{
			{"https://boards.example.com/jobs/1", "greenhouse"},
			{"https://www.linkedin.com/jobs/view/1", "linkedin"},
			{"https://workinstartups.com/details/1", "workinstartups"},
			{"https://www.linkedin.com/jobs/view/1", "linkedin"},
		} {
			posting.URL, posting.Source = listing.url, listing.source
			var err error
			if saved, _, err = st.SaveCanonical(t.Context(), posting); err != nil {
				t.Fatalf("SaveCanonical(%s) err = %v", listing.url, err)
			}
		}

		job, err := jobsearch.NewService(st, nil, jobsearchtest.NewNoopScoring()).Get(t.Context(), userID, saved.ID)
		if err != nil {
			t.Fatalf("Get() err = %v", err)
		}
		want := []dto.JobListing{
			{Source: "greenhouse", URL: "https://boards.example.com/jobs/1"},
			{Source: "linkedin", URL: "https://www.linkedin.com/jobs/view/1"},
			{Source: "workinstartups", URL: "https://workinstartups.com/details/1"},
		}
		if diff := cmp.Diff(want, job.Listings, cmpopts.IgnoreFields(dto.JobListing{}, "FirstSeenAt")); diff != "" {
			t.Errorf("Get().Listings (-want +got):\n%s", diff)
		}
		if job.URL != want[0].URL {
			t.Errorf("Get().URL = %q, want the primary %q", job.URL, want[0].URL)
		}
	})
}

func TestListHidesExcludedCompanies(t *testing.T) {
	st := jobsearchtest.NewFakeStore()
	for _, job := range []dto.Job{
		{Title: "Role", URL: "https://example.com/acme", CompanySlug: "acme"},
		{Title: "Role", URL: "https://example.com/globex", CompanySlug: "globex"},
	} {
		job.ScrapedAt = time.Now()
		if _, _, err := st.SaveCanonical(t.Context(), job); err != nil {
			t.Fatalf("SaveCanonical(%s) err = %v", job.URL, err)
		}
	}
	scoring := jobsearchtest.NewNoopScoring()
	scoring.SeedSearchConfig(dto.SearchConfig{UserID: userID, ExcludedCompanies: []string{"Acme Corp"}})

	page, err := jobsearch.NewService(st, nil, scoring).List(t.Context(), userID, dto.JobsQuery{})
	if err != nil {
		t.Fatalf("List() err = %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].CompanySlug != "globex" {
		t.Errorf("List() = %+v, want only the globex job", page.Items)
	}
}

func TestOverview(t *testing.T) {
	t.Parallel()
	svc := jobsearch.NewService(jobsearchtest.NewFakeStore(), nil, jobsearchtest.NewNoopScoring())

	t.Run("rejects a malformed day_start", func(t *testing.T) {
		t.Parallel()
		_, err := svc.Overview(t.Context(), userID, dto.JobOverviewQuery{DayStart: "today"})
		if !apperr.IsKind(err, apperr.KindInvalid) {
			t.Errorf("Overview() err = %v, want kind %v", err, apperr.KindInvalid)
		}
	})

	t.Run("rejects an unknown ranked value", func(t *testing.T) {
		t.Parallel()
		_, err := svc.Overview(t.Context(), userID, dto.JobOverviewQuery{Ranked: "false"})
		if !apperr.IsKind(err, apperr.KindInvalid) {
			t.Errorf("Overview() err = %v, want kind %v", err, apperr.KindInvalid)
		}
	})

	t.Run("an empty search returns zero stats and an empty, non-nil top list", func(t *testing.T) {
		t.Parallel()
		got, err := svc.Overview(t.Context(), userID, dto.JobOverviewQuery{})
		if err != nil {
			t.Fatalf("Overview() err = %v", err)
		}
		want := dto.JobOverview{Top: []dto.Job{}}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Overview() (-want +got):\n%s", diff)
		}
	})
}

func TestInsights(t *testing.T) {
	t.Parallel()

	t.Run("rejects an unknown time zone", func(t *testing.T) {
		t.Parallel()
		svc := jobsearch.NewService(jobsearchtest.NewFakeStore(), nil, jobsearchtest.NewNoopScoring())
		_, err := svc.Insights(t.Context(), userID, dto.JobInsightsQuery{TZ: "Mars/Olympus"})
		if !apperr.IsKind(err, apperr.KindInvalid) {
			t.Errorf("Insights() err = %v, want kind %v", err, apperr.KindInvalid)
		}
	})

	t.Run("no jobs returns empty non-nil series", func(t *testing.T) {
		t.Parallel()
		svc := jobsearch.NewService(jobsearchtest.NewFakeStore(), nil, jobsearchtest.NewNoopScoring())
		got, err := svc.Insights(t.Context(), userID, dto.JobInsightsQuery{})
		if err != nil {
			t.Fatalf("Insights() err = %v", err)
		}
		want := dto.JobInsights{
			Daily: []dto.JobInsightDay{}, Sources: []dto.JobInsightSource{},
			ScoreBands: []int{0, 0, 0, 0, 0}, FoundBy: []dto.JobInsightFinder{},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Insights() (-want +got):\n%s", diff)
		}
	})

	t.Run("folds groups into the chart series", func(t *testing.T) {
		t.Parallel()
		st := jobsearchtest.NewFakeStore()
		st.SeedInsightGroups(
			dto.JobInsightGroup{Day: "2026-03-02", Source: "lever", ScoreBand: 4, FoundBy: "wellfound", Jobs: 2, Scored: 2, ScoreSum: 171},
			dto.JobInsightGroup{Day: "2026-03-01", Source: "lever", ScoreBand: 1, Jobs: 1, Scored: 1, ScoreSum: 30},
			dto.JobInsightGroup{Day: "2026-03-01", Source: "ashby", ScoreBand: -1, Jobs: 1},
		)
		got, err := jobsearch.NewService(st, nil, jobsearchtest.NewNoopScoring()).Insights(t.Context(), userID, dto.JobInsightsQuery{TZ: "Europe/London"})
		if err != nil {
			t.Fatalf("Insights() err = %v", err)
		}
		want := dto.JobInsights{
			Total: 4, Scored: 3,
			Daily:      []dto.JobInsightDay{{Day: "2026-03-01", Count: 2}, {Day: "2026-03-02", Count: 2}},
			Sources:    []dto.JobInsightSource{{Source: "lever", Count: 3, Scored: 3, AvgScore: 67}, {Source: "ashby", Count: 1}},
			ScoreBands: []int{0, 1, 0, 0, 2},
			FoundBy:    []dto.JobInsightFinder{{Name: "wellfound", Count: 2}},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Insights() (-want +got):\n%s", diff)
		}
	})
}

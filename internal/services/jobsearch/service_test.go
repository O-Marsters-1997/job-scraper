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
				if len(page.Items) != tt.wantItems || page.Total != 3 {
					t.Errorf("List(%+v) = %d items, total %d, want %d items, total 3", tt.query, len(page.Items), page.Total, tt.wantItems)
				}
			})
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

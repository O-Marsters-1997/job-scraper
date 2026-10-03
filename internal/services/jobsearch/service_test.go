package jobsearch_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

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
			ScrapedAt: time.Date(2026, 1, i+1, 0, 0, 0, 0, time.UTC), UpdatedAt: time.Now(),
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
			{"limit too large", dto.JobsQuery{Limit: "9999"}},
			{"cursor not base64", dto.JobsQuery{Cursor: "not-base64"}},
			{"unknown availability", dto.JobsQuery{Availability: "unknown"}},
			{"negative since_days", dto.JobsQuery{SinceDays: "-1"}},
			{"huge since_days", dto.JobsQuery{SinceDays: "2147483647"}},
			{"non-numeric since_days", dto.JobsQuery{SinceDays: "week"}},
		}
		svc := jobsearch.NewService(jobsearchtest.NewFakeStore(), nil)
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := svc.List(t.Context(), userID, tt.query)
				if !apperr.IsKind(err, apperr.KindInvalid) {
					t.Errorf("List(%+v) err = %v, want kind %v", tt.query, err, apperr.KindInvalid)
				}
			})
		}
	})

	t.Run("windows on updated_at, defaulting to 90 days, 0 for none", func(t *testing.T) {
		st := jobsearchtest.NewFakeStore()
		now := time.Now()
		for url, age := range map[string]int{"https://example.com/fresh": 89, "https://example.com/stale": 91} {
			job := dto.Job{Title: "Role", URL: url, UpdatedAt: now.AddDate(0, 0, -age), ScrapedAt: now}
			if _, _, err := st.SaveCanonical(t.Context(), job); err != nil {
				t.Fatalf("SaveCanonical(%s) err = %v", url, err)
			}
		}
		svc := jobsearch.NewService(st, nil)

		for _, tt := range []struct {
			query dto.JobsQuery
			want  int
		}{
			{dto.JobsQuery{}, 1},
			{dto.JobsQuery{SinceDays: "0"}, 2},
			{dto.JobsQuery{SinceDays: "7"}, 0},
		} {
			page, err := svc.List(t.Context(), userID, tt.query)
			if err != nil {
				t.Fatalf("List(%+v) err = %v", tt.query, err)
			}
			if len(page.Items) != tt.want {
				t.Errorf("List(%+v) = %d items, want %d", tt.query, len(page.Items), tt.want)
			}
		}
	})

	t.Run("paginates", func(t *testing.T) {
		svc := jobsearch.NewService(seedJobs(t, 3), nil)

		page, err := svc.List(t.Context(), userID, dto.JobsQuery{Limit: "2"})
		if err != nil {
			t.Fatalf("List(first) err = %v", err)
		}
		if len(page.Items) != 2 || page.NextCursor == "" {
			t.Fatalf("first page = %+v, want 2 items and a cursor", page)
		}

		next, err := svc.List(t.Context(), userID, dto.JobsQuery{Limit: "2", Cursor: page.NextCursor})
		if err != nil {
			t.Fatalf("List(next) err = %v", err)
		}
		if len(next.Items) != 1 || next.NextCursor != "" {
			t.Errorf("next page = %+v, want 1 item and no cursor", next)
		}
	})
}

func TestGet(t *testing.T) {
	svc := jobsearch.NewService(jobsearchtest.NewFakeStore(), nil)
	_, err := svc.Get(t.Context(), userID, "missing")
	if !apperr.IsKind(err, apperr.KindNotFound) {
		t.Fatalf("Get(missing) err = %v, want kind %v", err, apperr.KindNotFound)
	}
}

func TestListScored(t *testing.T) {
	seedScored := func(t *testing.T, n int) *jobsearchtest.FakeStore {
		t.Helper()
		st := jobsearchtest.NewFakeStore()
		for i := range n {
			score := 100 - i
			job := dto.Job{Title: "Role", URL: "https://example.com/" + strconv.Itoa(i), UpdatedAt: time.Now(), SuitabilityScore: &score}
			if _, _, err := st.SaveCanonical(t.Context(), job); err != nil {
				t.Fatalf("SaveCanonical(%s) err = %v", job.URL, err)
			}
		}
		return st
	}
	list := func(t *testing.T, st *jobsearchtest.FakeStore, user string) []dto.Job {
		t.Helper()
		jobs, err := jobsearch.NewService(st, nil).ListScored(t.Context(), user)
		if err != nil {
			t.Fatalf("ListScored() err = %v", err)
		}
		return jobs
	}

	t.Run("flags one bottom-half job per ten, one in each block of ten slots", func(t *testing.T) {
		st := seedScored(t, 45)
		jobs := list(t, st, userID)
		if len(jobs) != 45 {
			t.Fatalf("ListScored() = %d jobs, want 45", len(jobs))
		}
		for block := range 4 {
			var flagged []int
			for i, job := range jobs[block*10 : block*10+10] {
				if job.Wildcard {
					flagged = append(flagged, i)
					if *job.SuitabilityScore > 100-23 {
						t.Errorf("block %d wildcard score = %d, want from the bottom half", block, *job.SuitabilityScore)
					}
				}
			}
			if len(flagged) != 1 {
				t.Errorf("block %d wildcards at %v, want exactly one", block, flagged)
			}
		}
		for _, job := range jobs[40:] {
			if job.Wildcard {
				t.Errorf("trailing partial block has wildcard %q", job.URL)
			}
		}
	})

	t.Run("keeps every job once, the rest in score order", func(t *testing.T) {
		st := seedScored(t, 45)
		jobs := list(t, st, userID)
		seen := map[string]bool{}
		prev := 101
		for _, job := range jobs {
			if seen[job.URL] {
				t.Errorf("job %s repeated", job.URL)
			}
			seen[job.URL] = true
			if job.Wildcard {
				continue
			}
			if *job.SuitabilityScore > prev {
				t.Errorf("score %d after %d, want non-wildcards best first", *job.SuitabilityScore, prev)
			}
			prev = *job.SuitabilityScore
		}
		if len(seen) != 45 {
			t.Errorf("distinct jobs = %d, want 45", len(seen))
		}
	})

	t.Run("is stable across calls", func(t *testing.T) {
		st := seedScored(t, 45)
		first := list(t, st, userID)
		if diff := cmp.Diff(first, list(t, st, userID)); diff != "" {
			t.Errorf("ListScored() changed between calls (-first +second):\n%s", diff)
		}
	})

	t.Run("short lists have no wildcard", func(t *testing.T) {
		for _, job := range list(t, seedScored(t, 9), userID) {
			if job.Wildcard {
				t.Errorf("job %s flagged in a list under ten", job.URL)
			}
		}
	})
}

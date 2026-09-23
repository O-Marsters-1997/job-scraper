package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

var baseJob = dto.Job{
	Title:       "Software Engineer",
	Location:    "London",
	URL:         "https://example.com/jobs/1",
	CompanySlug: "example",
	Source:      "greenhouse",
	UpdatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
}

// jobCmpOpts compares dto.Job values, ignoring DB-generated fields (ID, ScrapedAt).
var jobCmpOpts = cmp.Options{
	cmpopts.IgnoreFields(dto.Job{}, "ID", "ScrapedAt"),
	cmp.Comparer(func(x, y time.Time) bool {
		return x.Equal(y)
	}),
}

func findByURL(jobs []dto.Job, url string) (dto.Job, bool) {
	for _, j := range jobs {
		if j.URL == url {
			return j, true
		}
	}
	return dto.Job{}, false
}

func TestSave_Single(t *testing.T) {
	t.Run("insert", func(t *testing.T) {
		truncate(t)
		ctx := context.Background()

		if _, err := testDB.Save(ctx, []dto.Job{baseJob}); err != nil {
			t.Fatalf("Save: %v", err)
		}

		jobs, err := testDB.List(ctx, "")
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		got, ok := findByURL(jobs, baseJob.URL)
		if !ok {
			t.Fatal("saved job not found in List")
		}
		if diff := cmp.Diff(baseJob, got, jobCmpOpts...); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("updates on conflict", func(t *testing.T) {
		truncate(t)
		ctx := context.Background()

		if _, err := testDB.Save(ctx, []dto.Job{baseJob}); err != nil {
			t.Fatalf("first Save: %v", err)
		}

		updated := baseJob
		updated.Title = "Senior Software Engineer"
		updated.Location = "Remote"
		updated.UpdatedAt = time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

		if _, err := testDB.Save(ctx, []dto.Job{updated}); err != nil {
			t.Fatalf("second Save: %v", err)
		}

		jobs, err := testDB.List(ctx, "")
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		got, ok := findByURL(jobs, baseJob.URL)
		if !ok {
			t.Fatal("updated job not found in List")
		}
		if diff := cmp.Diff(updated, got, jobCmpOpts...); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestSave_Batch(t *testing.T) {
	t.Run("insert", func(t *testing.T) {
		truncate(t)
		ctx := context.Background()

		jobs := []dto.Job{
			baseJob,
			{
				Title:       "Product Manager",
				Location:    "New York",
				URL:         "https://example.com/jobs/2",
				CompanySlug: "example",
				Source:      "greenhouse",
				UpdatedAt:   time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				Title:       "Designer",
				Location:    "Berlin",
				URL:         "https://example.com/jobs/3",
				CompanySlug: "example",
				Source:      "greenhouse",
				UpdatedAt:   time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			},
		}

		if _, err := testDB.Save(ctx, jobs); err != nil {
			t.Fatalf("Save: %v", err)
		}

		got, err := testDB.List(ctx, "")
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != len(jobs) {
			t.Errorf("row count: got %d, want %d", len(got), len(jobs))
		}
	})

	t.Run("updates on conflict", func(t *testing.T) {
		truncate(t)
		ctx := context.Background()

		if _, err := testDB.Save(ctx, []dto.Job{baseJob}); err != nil {
			t.Fatalf("first Save: %v", err)
		}

		updated := baseJob
		updated.Title = "Staff Engineer"
		updated.Location = "Remote"
		updated.UpdatedAt = time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)

		if _, err := testDB.Save(ctx, []dto.Job{updated}); err != nil {
			t.Fatalf("second Save: %v", err)
		}

		jobs, err := testDB.List(ctx, "")
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(jobs) != 1 {
			t.Errorf("row count: got %d, want 1 (upsert must not duplicate)", len(jobs))
		}
		got, ok := findByURL(jobs, baseJob.URL)
		if !ok {
			t.Fatal("updated job not found in List")
		}
		if diff := cmp.Diff(updated, got, jobCmpOpts...); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestSave_Empty(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	if _, err := testDB.Save(ctx, nil); err != nil {
		t.Errorf("Save(nil): %v", err)
	}
	if _, err := testDB.Save(ctx, []dto.Job{}); err != nil {
		t.Errorf("Save([]): %v", err)
	}
}

func TestSaveCanonical_AliasesAndReplay(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	first := baseJob
	first.BoardID = "11111111-1111-1111-1111-111111111111"
	first.ProviderPostingID = "posting-1"
	first.URL = "https://example.com/jobs/1?ref=board"

	saved, status, err := testDB.SaveCanonical(ctx, first)
	if err != nil || status != "new" || saved.ID == "" {
		t.Fatalf("first save: status=%q job=%+v err=%v", status, saved, err)
	}
	alias := first
	alias.URL = "https://example.com/jobs/1?ref=partner"
	savedAgain, status, err := testDB.SaveCanonical(ctx, alias)
	if err != nil || status != "unchanged" || savedAgain.ID != saved.ID {
		t.Fatalf("alias: status=%q job=%+v err=%v", status, savedAgain, err)
	}
	replayed, status, err := testDB.SaveCanonical(ctx, alias)
	if err != nil || status != "unchanged" || replayed.ID != saved.ID {
		t.Fatalf("replay: status=%q job=%+v err=%v", status, replayed, err)
	}
	jobs, err := testDB.List(ctx, "")
	if err != nil || len(jobs) != 1 {
		t.Fatalf("list: count=%d err=%v", len(jobs), err)
	}
	if jobs[0].BoardID != first.BoardID || jobs[0].ProviderPostingID != first.ProviderPostingID || jobs[0].ContentFingerprint == "" {
		t.Fatalf("canonical identity missing from read: %+v", jobs[0])
	}
}

func TestSaveCanonical_ContentChangeAndDistinctBoard(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	first := baseJob
	first.BoardID = "11111111-1111-1111-1111-111111111111"
	first.ProviderPostingID = "posting-1"
	first.URL = "https://example.com/jobs/1"
	saved, _, err := testDB.SaveCanonical(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	changed := first
	changed.Title = "Senior Engineer"
	updated, status, err := testDB.SaveCanonical(ctx, changed)
	if err != nil || status != "changed" || updated.ID != saved.ID {
		t.Fatalf("changed: status=%q job=%+v err=%v", status, updated, err)
	}
	unchanged, status, err := testDB.SaveCanonical(ctx, changed)
	if err != nil || status != "unchanged" || unchanged.ID != saved.ID || unchanged.ContentFingerprint != updated.ContentFingerprint {
		t.Fatalf("changed replay: status=%q job=%+v err=%v", status, unchanged, err)
	}
	other := first
	other.BoardID = "22222222-2222-2222-2222-222222222222"
	other.URL = "https://other.example.com/jobs/1"
	distinct, status, err := testDB.SaveCanonical(ctx, other)
	if err != nil || status != "new" || distinct.ID == saved.ID {
		t.Fatalf("distinct board: status=%q job=%+v err=%v", status, distinct, err)
	}
}

func TestSaveCanonical_ConflictingBoardCannotClaimURL(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	first := baseJob
	first.BoardID = "11111111-1111-1111-1111-111111111111"
	first.ProviderPostingID = "posting-1"
	first.URL = "https://example.com/jobs/1"
	saved, _, err := testDB.SaveCanonical(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	other := first
	other.BoardID = "22222222-2222-2222-2222-222222222222"
	if _, _, err := testDB.SaveCanonical(ctx, other); err == nil {
		t.Fatal("expected conflicting board URL to be rejected")
	}
	jobs, err := testDB.List(ctx, "")
	if err != nil || len(jobs) != 1 || jobs[0].ID != saved.ID {
		t.Fatalf("canonical jobs changed after conflict: %+v, %v", jobs, err)
	}
}

func TestSaveCanonical_BackfillsLegacyJobFingerprint(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	if _, err := testDB.Save(ctx, []dto.Job{baseJob}); err != nil {
		t.Fatal(err)
	}
	_, status, err := testDB.SaveCanonical(ctx, baseJob)
	if err != nil || status != "unchanged" {
		t.Fatalf("legacy replay: status=%q err=%v", status, err)
	}
	jobs, err := testDB.List(ctx, "")
	if err != nil || len(jobs) != 1 || jobs[0].ContentFingerprint == "" {
		t.Fatalf("legacy fingerprint missing: jobs=%+v err=%v", jobs, err)
	}
}

func TestNewURLs(t *testing.T) {
	t.Run("filters existing URLs", func(t *testing.T) {
		truncate(t)
		ctx := context.Background()

		if _, err := testDB.Save(ctx, []dto.Job{baseJob}); err != nil {
			t.Fatalf("Save: %v", err)
		}

		newJobURL := "https://example.com/jobs/new"
		got, err := testDB.NewURLs(ctx, []string{baseJob.URL, newJobURL})
		if err != nil {
			t.Fatalf("NewURLs: %v", err)
		}
		if len(got) != 1 || got[0] != newJobURL {
			t.Errorf("want [%q], got %v", newJobURL, got)
		}
	})

	t.Run("all new URLs returned unchanged", func(t *testing.T) {
		truncate(t)
		ctx := context.Background()

		urls := []string{"https://example.com/a", "https://example.com/b"}
		got, err := testDB.NewURLs(ctx, urls)
		if err != nil {
			t.Fatalf("NewURLs: %v", err)
		}
		if len(got) != len(urls) {
			t.Errorf("want %d URLs, got %d", len(urls), len(got))
		}
	})
}

func TestOpenJobURLsForBoard_MarkJobsClosed(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	source := "greenhouse"
	companySlug := "acme"
	jobs := []dto.Job{
		{Title: "A", URL: "https://boards.example.com/acme/1", CompanySlug: companySlug, Source: source, UpdatedAt: time.Now()},
		{Title: "B", URL: "https://boards.example.com/acme/2", CompanySlug: companySlug, Source: source, UpdatedAt: time.Now()},
		{Title: "C", URL: "https://boards.example.com/acme/3", CompanySlug: companySlug, Source: source, UpdatedAt: time.Now()},
	}
	if _, err := testDB.Save(ctx, jobs); err != nil {
		t.Fatalf("Save: %v", err)
	}

	open, err := testDB.OpenJobURLsForBoard(ctx, source, companySlug)
	if err != nil {
		t.Fatalf("OpenJobURLsForBoard: %v", err)
	}
	if len(open) != 3 {
		t.Fatalf("want 3 open urls, got %d", len(open))
	}

	// Only jobs 1 and 2 present in this run; job 3 disappeared.
	currentRun := map[string]bool{jobs[0].URL: true, jobs[1].URL: true}
	var toClose []string
	for _, url := range open {
		if !currentRun[url] {
			toClose = append(toClose, url)
		}
	}
	if len(toClose) != 1 || toClose[0] != jobs[2].URL {
		t.Fatalf("want to close [%q], got %v", jobs[2].URL, toClose)
	}

	if err := testDB.MarkJobsClosed(ctx, toClose); err != nil {
		t.Fatalf("MarkJobsClosed: %v", err)
	}

	open, err = testDB.OpenJobURLsForBoard(ctx, source, companySlug)
	if err != nil {
		t.Fatalf("OpenJobURLsForBoard (after close): %v", err)
	}
	if len(open) != 2 {
		t.Fatalf("want 2 open urls after closing job 3, got %v", open)
	}
	for _, url := range open {
		if url == jobs[2].URL {
			t.Fatalf("closed job %q still reported open", jobs[2].URL)
		}
	}

	// Re-scraping the closed job (fresh Save) reopens it for free.
	reopened := jobs[2]
	reopened.UpdatedAt = time.Now()
	if _, err := testDB.Save(ctx, []dto.Job{reopened}); err != nil {
		t.Fatalf("Save (reopen): %v", err)
	}
	open, err = testDB.OpenJobURLsForBoard(ctx, source, companySlug)
	if err != nil {
		t.Fatalf("OpenJobURLsForBoard (after reopen): %v", err)
	}
	if len(open) != 3 {
		t.Fatalf("want 3 open urls after re-saving job 3, got %v", open)
	}
}

func TestList(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	jobs := []dto.Job{
		baseJob,
		{
			Title:       "Other Role",
			URL:         "https://example.com/jobs/other",
			CompanySlug: "example",
			Source:      "greenhouse",
			UpdatedAt:   time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	if _, err := testDB.Save(ctx, jobs); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := testDB.List(ctx, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != len(jobs) {
		t.Errorf("want %d jobs, got %d", len(jobs), len(got))
	}
}

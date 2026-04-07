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

		if err := testDB.Save(ctx, []dto.Job{baseJob}); err != nil {
			t.Fatalf("Save: %v", err)
		}

		jobs, err := testDB.List(ctx)
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

		if err := testDB.Save(ctx, []dto.Job{baseJob}); err != nil {
			t.Fatalf("first Save: %v", err)
		}

		updated := baseJob
		updated.Title = "Senior Software Engineer"
		updated.Location = "Remote"
		updated.UpdatedAt = time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

		if err := testDB.Save(ctx, []dto.Job{updated}); err != nil {
			t.Fatalf("second Save: %v", err)
		}

		jobs, err := testDB.List(ctx)
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

		if err := testDB.Save(ctx, jobs); err != nil {
			t.Fatalf("Save: %v", err)
		}

		got, err := testDB.List(ctx)
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

		if err := testDB.Save(ctx, []dto.Job{baseJob}); err != nil {
			t.Fatalf("first Save: %v", err)
		}

		updated := baseJob
		updated.Title = "Staff Engineer"
		updated.Location = "Remote"
		updated.UpdatedAt = time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)

		if err := testDB.Save(ctx, []dto.Job{updated}); err != nil {
			t.Fatalf("second Save: %v", err)
		}

		jobs, err := testDB.List(ctx)
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

	if err := testDB.Save(ctx, nil); err != nil {
		t.Errorf("Save(nil): %v", err)
	}
	if err := testDB.Save(ctx, []dto.Job{}); err != nil {
		t.Errorf("Save([]): %v", err)
	}
}

func TestNewURLs(t *testing.T) {
	t.Run("filters existing URLs", func(t *testing.T) {
		truncate(t)
		ctx := context.Background()

		if err := testDB.Save(ctx, []dto.Job{baseJob}); err != nil {
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

	if err := testDB.Save(ctx, jobs); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := testDB.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != len(jobs) {
		t.Errorf("want %d jobs, got %d", len(jobs), len(got))
	}
}

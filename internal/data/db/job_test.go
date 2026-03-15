package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
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

// jobCmpOpts are the cmp options used when comparing pgsqlc.Job values.
// ID and ScrapedAt are DB-generated so they are excluded from comparisons.
// pgtype.Timestamptz is compared by its underlying time value.
var jobCmpOpts = cmp.Options{
	cmpopts.IgnoreFields(pgsqlc.Job{}, "ID", "ScrapedAt"),
	cmp.Comparer(func(x, y pgtype.Timestamptz) bool {
		return x.Valid == y.Valid && x.Time.Equal(y.Time)
	}),
}

// jobFromSource builds the expected pgsqlc.Job from a sources.Job for use in
// cmp.Diff assertions.
func jobFromSource(j dto.Job) pgsqlc.Job {
	return pgsqlc.Job{
		Title:       j.Title,
		Location:    j.Location,
		Url:         j.URL,
		CompanySlug: j.CompanySlug,
		Source:      j.Source,
		UpdatedAt:   pgtype.Timestamptz{Time: j.UpdatedAt, Valid: true},
	}
}

func TestUpsertJob(t *testing.T) {
	t.Run("insert", func(t *testing.T) {
		truncate(t)
		ctx := context.Background()

		if err := testDB.UpsertJob(ctx, baseJob); err != nil {
			t.Fatalf("UpsertJob: %v", err)
		}

		got, err := pgsqlc.New(testDB.Pool()).GetJobByURL(ctx, baseJob.URL)
		if err != nil {
			t.Fatalf("GetJobByURL: %v", err)
		}

		if diff := cmp.Diff(jobFromSource(baseJob), got, jobCmpOpts...); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("updates on conflict", func(t *testing.T) {
		truncate(t)
		ctx := context.Background()

		if err := testDB.UpsertJob(ctx, baseJob); err != nil {
			t.Fatalf("first UpsertJob: %v", err)
		}

		updated := baseJob
		updated.Title = "Senior Software Engineer"
		updated.Location = "Remote"
		updated.UpdatedAt = time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

		if err := testDB.UpsertJob(ctx, updated); err != nil {
			t.Fatalf("second UpsertJob: %v", err)
		}

		got, err := pgsqlc.New(testDB.Pool()).GetJobByURL(ctx, baseJob.URL)
		if err != nil {
			t.Fatalf("GetJobByURL: %v", err)
		}

		if diff := cmp.Diff(jobFromSource(updated), got, jobCmpOpts...); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestUpsertJobs(t *testing.T) {
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

		if err := testDB.UpsertJobs(ctx, jobs); err != nil {
			t.Fatalf("UpsertJobs: %v", err)
		}

		rows, err := pgsqlc.New(testDB.Pool()).ListJobs(ctx)
		if err != nil {
			t.Fatalf("ListJobs: %v", err)
		}
		if len(rows) != len(jobs) {
			t.Errorf("row count: got %d, want %d", len(rows), len(jobs))
		}
	})

	t.Run("updates on conflict", func(t *testing.T) {
		truncate(t)
		ctx := context.Background()

		if err := testDB.UpsertJobs(ctx, []dto.Job{baseJob}); err != nil {
			t.Fatalf("first UpsertJobs: %v", err)
		}

		updated := baseJob
		updated.Title = "Staff Engineer"
		updated.Location = "Remote"
		updated.UpdatedAt = time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)

		if err := testDB.UpsertJobs(ctx, []dto.Job{updated}); err != nil {
			t.Fatalf("second UpsertJobs: %v", err)
		}

		got, err := pgsqlc.New(testDB.Pool()).GetJobByURL(ctx, baseJob.URL)
		if err != nil {
			t.Fatalf("GetJobByURL: %v", err)
		}

		if diff := cmp.Diff(jobFromSource(updated), got, jobCmpOpts...); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}

		rows, err := pgsqlc.New(testDB.Pool()).ListJobs(ctx)
		if err != nil {
			t.Fatalf("ListJobs: %v", err)
		}
		if len(rows) != 1 {
			t.Errorf("row count: got %d, want 1 (upsert must not duplicate)", len(rows))
		}
	})

	t.Run("empty", func(t *testing.T) {
		truncate(t)
		ctx := context.Background()

		if err := testDB.UpsertJobs(ctx, []dto.Job{}); err != nil {
			t.Fatalf("UpsertJobs with empty slice: %v", err)
		}
	})
}

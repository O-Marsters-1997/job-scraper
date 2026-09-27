package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func insertJobScore(t testing.TB, ctx context.Context, jobID, userID string) {
	t.Helper()
	if _, err := testDB.Pool().Exec(ctx, "INSERT INTO job_scores (job_id, user_id) VALUES ($1, $2)", jobID, userID); err != nil {
		t.Fatalf("insert job_scores: %v", err)
	}
}

func closeJob(t testing.TB, ctx context.Context, jobID string) {
	t.Helper()
	if _, err := testDB.Pool().Exec(ctx, "UPDATE jobs SET closed_at = NOW() WHERE id = $1", jobID); err != nil {
		t.Fatalf("close job: %v", err)
	}
}

func findOption(options []dto.ScoringOption, id string) (dto.ScoringOption, bool) {
	for _, o := range options {
		if o.ID == id {
			return o, true
		}
	}
	return dto.ScoringOption{}, false
}

func TestListScoringOptions(t *testing.T) {
	ctx := context.Background()

	dims := []dto.Dimension{
		dto.DimensionTech, dto.DimensionRole, dto.DimensionDomain,
		dto.DimensionSeniority, dto.DimensionWork, dto.DimensionStage,
	}
	for _, dim := range dims {
		id := "test:" + string(dim)
		if _, err := testDB.Pool().Exec(ctx, `
			INSERT INTO scoring_options (id, dimension, label, question) VALUES ($1, $2, $1, $1)
		`, id, dim); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if _, err := testDB.Pool().Exec(ctx, `DELETE FROM scoring_options WHERE id = $1`, id); err != nil {
				t.Fatalf("cleanup: %v", err)
			}
		})
	}

	options, err := testDB.ListScoringOptions(ctx)
	if err != nil {
		t.Fatal(err)
	}

	byDimension := make(map[dto.Dimension]int)
	for _, o := range options {
		byDimension[o.Dimension]++
	}
	for _, dim := range dims {
		if byDimension[dim] == 0 {
			t.Errorf("dimension %q has no options", dim)
		}
	}
}

func TestListScoringOptions_RoundTripsEnumAndRetiredAt(t *testing.T) {
	ctx := context.Background()

	_, err := testDB.Pool().Exec(ctx, `
		INSERT INTO scoring_options (id, dimension, label, question, retired_at)
		VALUES ('test:round-trip', 'seniority', 'Round trip', 'Does this round trip?', NOW())
	`)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := testDB.Pool().Exec(ctx, `DELETE FROM scoring_options WHERE id = 'test:round-trip'`); err != nil {
			t.Fatalf("cleanup: %v", err)
		}
	})

	options, err := testDB.ListScoringOptions(ctx)
	if err != nil {
		t.Fatal(err)
	}

	var found *dto.ScoringOption
	for i, o := range options {
		if o.ID == "test:round-trip" {
			found = &options[i]
			break
		}
	}
	if found == nil {
		t.Fatal("round-trip option not found")
	}
	if found.Dimension != dto.DimensionSeniority {
		t.Errorf("dimension = %q, want %q", found.Dimension, dto.DimensionSeniority)
	}
	if found.RetiredAt == nil {
		t.Error("retired_at should round-trip as non-nil")
	}
}

func TestAddScoringOption(t *testing.T) {
	t.Run("inserts the option", func(t *testing.T) {
		truncate(t)
		ctx := context.Background()

		if err := testDB.AddScoringOption(ctx, "tech:zig", "tech", "Zig", "Does the role use Zig?"); err != nil {
			t.Fatalf("AddScoringOption: %v", err)
		}

		options, err := testDB.ListScoringOptions(ctx)
		if err != nil {
			t.Fatal(err)
		}
		got, ok := findOption(options, "tech:zig")
		if !ok {
			t.Fatal("inserted option not found")
		}
		if got.Dimension != dto.DimensionTech || got.Label != "Zig" || got.Question != "Does the role use Zig?" {
			t.Errorf("option = %+v, want dimension=tech label=Zig question=%q", got, "Does the role use Zig?")
		}
	})

	t.Run("queues a backfill effect for every non-closed job with a job_scores row", func(t *testing.T) {
		truncate(t)
		ctx := context.Background()

		user, err := testDB.CreateUser(ctx, "backfill-user", "hash", "")
		if err != nil {
			t.Fatal(err)
		}
		company, err := testDB.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "backfill-co", Name: "Backfill Co"})
		if err != nil {
			t.Fatal(err)
		}

		scoredJob := baseJob
		scoredJob.URL = "https://example.com/jobs/backfill-scored"
		scoredJob.CompanySlug = company.Slug
		scoredJob.CompanyID = company.ID
		savedScored, _, err := testDB.SaveCanonical(ctx, scoredJob)
		if err != nil {
			t.Fatal(err)
		}
		insertJobScore(t, ctx, savedScored.ID, user.ID)

		closedJob := baseJob
		closedJob.URL = "https://example.com/jobs/backfill-closed"
		closedJob.CompanySlug = company.Slug
		closedJob.CompanyID = company.ID
		savedClosed, _, err := testDB.SaveCanonical(ctx, closedJob)
		if err != nil {
			t.Fatal(err)
		}
		insertJobScore(t, ctx, savedClosed.ID, user.ID)
		closeJob(t, ctx, savedClosed.ID)

		unscoredJob := baseJob
		unscoredJob.URL = "https://example.com/jobs/backfill-unscored"
		unscoredJob.CompanySlug = company.Slug
		unscoredJob.CompanyID = company.ID
		savedUnscored, _, err := testDB.SaveCanonical(ctx, unscoredJob)
		if err != nil {
			t.Fatal(err)
		}

		if err := testDB.AddScoringOption(ctx, "tech:zig", "tech", "Zig", "Does the role use Zig?"); err != nil {
			t.Fatalf("AddScoringOption: %v", err)
		}

		if count := effectCountForJob(t, ctx, savedScored.ID); count != 1 {
			t.Errorf("scored job effects = %d, want 1", count)
		}
		if count := effectCountForJob(t, ctx, savedClosed.ID); count != 0 {
			t.Errorf("closed job effects = %d, want 0", count)
		}
		if count := effectCountForJob(t, ctx, savedUnscored.ID); count != 0 {
			t.Errorf("unscored job effects = %d, want 0", count)
		}
	})
}

func TestRewordScoringOption(t *testing.T) {
	t.Run("updates the question text", func(t *testing.T) {
		truncate(t)
		ctx := context.Background()

		if err := testDB.AddScoringOption(ctx, "tech:zig", "tech", "Zig", "Does the role use Zig?"); err != nil {
			t.Fatal(err)
		}
		if err := testDB.RewordScoringOption(ctx, "tech:zig", "Is the role built with Zig?"); err != nil {
			t.Fatalf("RewordScoringOption: %v", err)
		}

		options, err := testDB.ListScoringOptions(ctx)
		if err != nil {
			t.Fatal(err)
		}
		got, ok := findOption(options, "tech:zig")
		if !ok {
			t.Fatal("option not found")
		}
		if got.Question != "Is the role built with Zig?" {
			t.Errorf("question = %q, want %q", got.Question, "Is the role built with Zig?")
		}
	})

	t.Run("queues a backfill effect for a scored job", func(t *testing.T) {
		truncate(t)
		ctx := context.Background()

		user, err := testDB.CreateUser(ctx, "reword-user", "hash", "")
		if err != nil {
			t.Fatal(err)
		}
		company, err := testDB.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "reword-co", Name: "Reword Co"})
		if err != nil {
			t.Fatal(err)
		}
		job := baseJob
		job.URL = "https://example.com/jobs/reword"
		job.CompanySlug = company.Slug
		job.CompanyID = company.ID
		saved, _, err := testDB.SaveCanonical(ctx, job)
		if err != nil {
			t.Fatal(err)
		}
		insertJobScore(t, ctx, saved.ID, user.ID)

		if err := testDB.AddScoringOption(ctx, "tech:zig", "tech", "Zig", "Does the role use Zig?"); err != nil {
			t.Fatal(err)
		}
		if _, err := testDB.Pool().Exec(ctx, "DELETE FROM effect_outbox WHERE job_id = $1", saved.ID); err != nil {
			t.Fatal(err)
		}

		if err := testDB.RewordScoringOption(ctx, "tech:zig", "Is the role built with Zig?"); err != nil {
			t.Fatalf("RewordScoringOption: %v", err)
		}

		if count := effectCountForJob(t, ctx, saved.ID); count != 1 {
			t.Errorf("effects = %d, want 1", count)
		}
	})

	t.Run("errors when the option does not exist", func(t *testing.T) {
		truncate(t)
		ctx := context.Background()

		err := testDB.RewordScoringOption(ctx, "tech:missing", "Does this exist?")
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("err = %v, want data.ErrNotFound", err)
		}
	})
}

func TestRetireScoringOption(t *testing.T) {
	t.Run("sets retired_at", func(t *testing.T) {
		truncate(t)
		ctx := context.Background()

		if err := testDB.AddScoringOption(ctx, "tech:zig", "tech", "Zig", "Does the role use Zig?"); err != nil {
			t.Fatal(err)
		}
		if err := testDB.RetireScoringOption(ctx, "tech:zig"); err != nil {
			t.Fatalf("RetireScoringOption: %v", err)
		}

		options, err := testDB.ListScoringOptions(ctx)
		if err != nil {
			t.Fatal(err)
		}
		got, ok := findOption(options, "tech:zig")
		if !ok {
			t.Fatal("option not found")
		}
		if got.RetiredAt == nil {
			t.Error("retired_at should be set")
		}
	})

	t.Run("errors when the option does not exist", func(t *testing.T) {
		truncate(t)
		ctx := context.Background()

		err := testDB.RetireScoringOption(ctx, "tech:missing")
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("err = %v, want data.ErrNotFound", err)
		}
	})

	t.Run("errors when the option is already retired", func(t *testing.T) {
		truncate(t)
		ctx := context.Background()

		if err := testDB.AddScoringOption(ctx, "tech:zig", "tech", "Zig", "Does the role use Zig?"); err != nil {
			t.Fatal(err)
		}
		if err := testDB.RetireScoringOption(ctx, "tech:zig"); err != nil {
			t.Fatal(err)
		}
		if err := testDB.RetireScoringOption(ctx, "tech:zig"); !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("err = %v, want data.ErrNotFound", err)
		}
	})
}

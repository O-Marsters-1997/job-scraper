package db_test

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
)

func TestSourceTargetRunGenerationFencesStaleCompletion(t *testing.T) {
	ctx := context.Background()
	truncateCompanies(t)
	var userID string
	if err := testDB.Pool().QueryRow(ctx, "INSERT INTO users (username, password_hash) VALUES ('run-user', 'x') RETURNING id::text").Scan(&userID); err != nil {
		t.Fatal(err)
	}
	target, err := testDB.CreateSourceTarget(ctx, userID, "wis", "engineer", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	first, err := testDB.StartSourceTargetRun(ctx, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Enabled {
		t.Fatal("manual rerun did not enable target")
	}
	second, err := testDB.StartSourceTargetRun(ctx, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if first.RunID == second.RunID || first.RunID == "" {
		t.Fatalf("run IDs = %q, %q", first.RunID, second.RunID)
	}
	if _, err := testDB.TransitionSourceTargetRun(ctx, target.ID, first.RunID, "succeeded", ""); err != providers.ErrNotFound {
		t.Fatalf("stale completion error = %v", err)
	}
	current, err := testDB.TransitionSourceTargetRun(ctx, target.ID, second.RunID, "succeeded", "")
	if err != nil || current.RunStatus != "succeeded" {
		t.Fatalf("current completion = %+v, %v", current, err)
	}
}

func TestCreateSourceTargetWithRunIsQueuedInSameInsert(t *testing.T) {
	ctx := context.Background()
	truncateCompanies(t)
	var userID string
	if err := testDB.Pool().QueryRow(ctx, "INSERT INTO users (username, password_hash) VALUES ('atomic-run-user', 'x') RETURNING id::text").Scan(&userID); err != nil {
		t.Fatal(err)
	}
	target, err := testDB.CreateSourceTargetWithRun(ctx, userID, "wis", "engineer", true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if target.RunStatus != "queued" || target.RunID == "" {
		t.Fatalf("created run = %+v", target)
	}
	if _, err := testDB.Pool().Exec(ctx, "UPDATE source_targets SET updated_at = NOW() - INTERVAL '2 minutes' WHERE id = $1", target.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ClaimRecoverableSourceTarget(ctx, target.ID, target.RunID); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ClaimRecoverableSourceTarget(ctx, target.ID, target.RunID); err != providers.ErrNotFound {
		t.Fatalf("second recovery claim error = %v", err)
	}
}

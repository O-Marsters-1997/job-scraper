package db_test

import (
	"context"
	"testing"
	"time"
)

func insertJobForOpsState(t *testing.T, url string) string {
	t.Helper()
	ctx := context.Background()
	var jobID string
	if err := testDB.Pool().QueryRow(ctx,
		`INSERT INTO jobs (title, url, company_slug, source, updated_at)
		 VALUES ('Engineer', $1, 'acme', 'greenhouse', NOW()) RETURNING id`, url).Scan(&jobID); err != nil {
		t.Fatalf("insert job %s: %v", url, err)
	}
	return jobID
}

func insertEffectOutbox(t *testing.T, jobID, status string, createdAt time.Time) {
	t.Helper()
	if _, err := testDB.Pool().Exec(context.Background(),
		`INSERT INTO effect_outbox (job_id, fingerprint, status, created_at)
		 VALUES ($1, 'fp', $2, $3)`,
		jobID, status, createdAt); err != nil {
		t.Fatalf("insert effect_outbox (%s): %v", status, err)
	}
}

func TestOpsState(t *testing.T) {
	ctx := context.Background()
	truncate(t)

	now := time.Now()
	insertEffectOutbox(t, insertJobForOpsState(t, "https://example.com/ops-state/1"), "pending", now.Add(-2*time.Hour))
	insertEffectOutbox(t, insertJobForOpsState(t, "https://example.com/ops-state/2"), "running", now.Add(-10*time.Minute))
	insertEffectOutbox(t, insertJobForOpsState(t, "https://example.com/ops-state/3"), "failed", now)
	insertEffectOutbox(t, insertJobForOpsState(t, "https://example.com/ops-state/4"), "failed", now)
	insertEffectOutbox(t, insertJobForOpsState(t, "https://example.com/ops-state/5"), "done", now)

	state, err := testDB.OpsState(ctx)
	if err != nil {
		t.Fatalf("OpsState: %v", err)
	}
	if state.OutboxPending != 2 {
		t.Errorf("OutboxPending = %d, want 2", state.OutboxPending)
	}
	if state.OutboxFailed != 2 {
		t.Errorf("OutboxFailed = %d, want 2", state.OutboxFailed)
	}
	if state.OutboxOldestPendingAge < 115*time.Minute || state.OutboxOldestPendingAge > 125*time.Minute {
		t.Errorf("OutboxOldestPendingAge = %s, want ~2h", state.OutboxOldestPendingAge)
	}
}

func TestOpsStateNoPending(t *testing.T) {
	ctx := context.Background()
	truncate(t)

	state, err := testDB.OpsState(ctx)
	if err != nil {
		t.Fatalf("OpsState: %v", err)
	}
	if state.OutboxPending != 0 || state.OutboxFailed != 0 {
		t.Errorf("expected zero counts, got %+v", state)
	}
	if state.OutboxOldestPendingAge != 0 {
		t.Errorf("OutboxOldestPendingAge = %s, want 0", state.OutboxOldestPendingAge)
	}
}

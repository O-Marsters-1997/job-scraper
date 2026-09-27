package db_test

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestCompleteAnswerEffect_CommitsAnswersAndBothUsersScoresTogether(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	company, err := testDB.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "complete-co", Name: "Complete Co"})
	if err != nil {
		t.Fatal(err)
	}
	alice, err := testDB.CreateUser(ctx, "complete-alice", "hash", "")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := testDB.CreateUser(ctx, "complete-bob", "hash", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, uid := range []string{alice.ID, bob.ID} {
		if _, err := testDB.SetCompanyTracking(ctx, uid, company.ID, true, 360); err != nil {
			t.Fatal(err)
		}
	}
	job := baseJob
	job.URL = "https://example.com/jobs/complete"
	job.CompanySlug = company.Slug
	job.CompanyID = company.ID
	saved, _, err := testDB.SaveCanonical(ctx, job)
	if err != nil {
		t.Fatal(err)
	}
	effect, err := testDB.ClaimAnswerEffect(ctx)
	if err != nil {
		t.Fatal(err)
	}

	answers := map[string]dto.Answer{
		"hash-go": {PYes: 0.9, PNo: 0.05, PNotStated: 0.05, Confidence: 0.9},
	}
	scores := []dto.JobScore{
		{JobID: saved.ID, UserID: alice.ID, Score: 79, Rows: []dto.ScoreRow{{Key: "tech:go", Label: "Go", Stance: "nice", Resolved: "yes", Effect: "meets"}}},
		{JobID: saved.ID, UserID: bob.ID, Score: 21, Rows: []dto.ScoreRow{{Key: "tech:go", Label: "Go", Stance: "avoid", Resolved: "yes", Effect: "misses"}}},
	}
	savedUsers, err := testDB.CompleteAnswerEffect(ctx, effect, answers, scores)
	if err != nil {
		t.Fatalf("CompleteAnswerEffect: %v", err)
	}
	if len(savedUsers) != 2 {
		t.Fatalf("saved users = %v, want 2", savedUsers)
	}

	var answerCount int
	if err := testDB.Pool().QueryRow(ctx, "SELECT count(*) FROM option_answers WHERE job_id = $1", saved.ID).Scan(&answerCount); err != nil {
		t.Fatal(err)
	}
	if answerCount != 1 {
		t.Fatalf("option_answers rows = %d, want 1", answerCount)
	}

	var scoreCount int
	if err := testDB.Pool().QueryRow(ctx, "SELECT count(*) FROM job_scores WHERE job_id = $1", saved.ID).Scan(&scoreCount); err != nil {
		t.Fatal(err)
	}
	if scoreCount != 2 {
		t.Fatalf("job_scores rows = %d, want 2", scoreCount)
	}

	var status string
	if err := testDB.Pool().QueryRow(ctx, "SELECT status FROM effect_outbox WHERE id = $1", effect.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "done" {
		t.Fatalf("effect status = %q, want done", status)
	}
}

func TestCompleteAnswerEffect_PersistsHidden(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	user, err := testDB.CreateUser(ctx, "hidden-user", "hash", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.CreateSourceTarget(ctx, user.ID, "greenhouse", "hidden-company", true, nil); err != nil {
		t.Fatal(err)
	}
	job := baseJob
	job.URL = "https://example.com/jobs/hidden"
	job.CompanySlug = "hidden-company"
	saved, _, err := testDB.SaveCanonical(ctx, job)
	if err != nil {
		t.Fatal(err)
	}
	effect, err := testDB.ClaimAnswerEffect(ctx)
	if err != nil {
		t.Fatal(err)
	}

	scores := []dto.JobScore{{JobID: saved.ID, UserID: user.ID, Score: 30, Hidden: true}}
	if _, err := testDB.CompleteAnswerEffect(ctx, effect, nil, scores); err != nil {
		t.Fatalf("CompleteAnswerEffect: %v", err)
	}

	var hidden bool
	if err := testDB.Pool().QueryRow(ctx, "SELECT hidden FROM job_scores WHERE job_id = $1 AND user_id = $2", saved.ID, user.ID).Scan(&hidden); err != nil {
		t.Fatal(err)
	}
	if !hidden {
		t.Fatal("hidden = false, want true")
	}
}

func TestQueueUserBackfill_QueuesScoredOpenJobsOnlyOnce(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	user, err := testDB.CreateUser(ctx, "backfill-user", "hash", "")
	if err != nil {
		t.Fatal(err)
	}

	open := baseJob
	open.URL = "https://example.com/jobs/backfill-open"
	savedOpen, _, err := testDB.SaveCanonical(ctx, open)
	if err != nil {
		t.Fatal(err)
	}
	closed := baseJob
	closed.URL = "https://example.com/jobs/backfill-closed"
	savedClosed, _, err := testDB.SaveCanonical(ctx, closed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Pool().Exec(ctx, "UPDATE jobs SET closed_at = NOW() WHERE id = $1", savedClosed.ID); err != nil {
		t.Fatal(err)
	}
	unscored := baseJob
	unscored.URL = "https://example.com/jobs/backfill-unscored"
	if _, _, err := testDB.SaveCanonical(ctx, unscored); err != nil {
		t.Fatal(err)
	}

	for _, jobID := range []string{savedOpen.ID, savedClosed.ID} {
		if _, err := testDB.Pool().Exec(ctx,
			"INSERT INTO job_scores (job_id, user_id) VALUES ($1, $2)", jobID, user.ID); err != nil {
			t.Fatal(err)
		}
	}

	queued, err := testDB.QueueUserBackfill(ctx, user.ID)
	if err != nil {
		t.Fatalf("QueueUserBackfill: %v", err)
	}
	if queued != 1 {
		t.Fatalf("queued = %d, want 1 (only the open, scored job)", queued)
	}

	var openCount int
	if err := testDB.Pool().QueryRow(ctx, "SELECT count(*) FROM effect_outbox WHERE job_id = $1", savedOpen.ID).Scan(&openCount); err != nil {
		t.Fatal(err)
	}
	if openCount != 1 {
		t.Fatalf("effects for open job = %d, want 1", openCount)
	}
	var closedCount int
	if err := testDB.Pool().QueryRow(ctx, "SELECT count(*) FROM effect_outbox WHERE job_id = $1", savedClosed.ID).Scan(&closedCount); err != nil {
		t.Fatal(err)
	}
	if closedCount != 0 {
		t.Fatalf("effects for closed job = %d, want 0", closedCount)
	}

	queuedAgain, err := testDB.QueueUserBackfill(ctx, user.ID)
	if err != nil {
		t.Fatalf("QueueUserBackfill (again): %v", err)
	}
	if queuedAgain != 0 {
		t.Fatalf("queued on second call = %d, want 0 (already pending)", queuedAgain)
	}
}

func TestCompleteAnswerEffect_FingerprintChangeWritesNothing(t *testing.T) {
	truncate(t)
	ctx := context.Background()

	user, err := testDB.CreateUser(ctx, "stale-user", "hash", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.CreateSourceTarget(ctx, user.ID, "greenhouse", "stale-company", true, nil); err != nil {
		t.Fatal(err)
	}
	job := baseJob
	job.URL = "https://example.com/jobs/stale"
	job.CompanySlug = "stale-company"
	saved, _, err := testDB.SaveCanonical(ctx, job)
	if err != nil {
		t.Fatal(err)
	}
	effect, err := testDB.ClaimAnswerEffect(ctx)
	if err != nil {
		t.Fatal(err)
	}

	changed := job
	changed.Title = "Staff Engineer"
	if _, _, err := testDB.SaveCanonical(ctx, changed); err != nil {
		t.Fatal(err)
	}

	answers := map[string]dto.Answer{"hash-go": {PYes: 0.9, PNo: 0.05, PNotStated: 0.05}}
	scores := []dto.JobScore{{JobID: saved.ID, UserID: user.ID, Score: 79}}
	savedUsers, err := testDB.CompleteAnswerEffect(ctx, effect, answers, scores)
	if err != nil {
		t.Fatalf("CompleteAnswerEffect: %v", err)
	}
	if len(savedUsers) != 0 {
		t.Fatalf("saved users = %v, want none (fingerprint moved on)", savedUsers)
	}

	var answerCount, scoreCount int
	if err := testDB.Pool().QueryRow(ctx, "SELECT count(*) FROM option_answers WHERE job_id = $1", saved.ID).Scan(&answerCount); err != nil {
		t.Fatal(err)
	}
	if err := testDB.Pool().QueryRow(ctx, "SELECT count(*) FROM job_scores WHERE job_id = $1", saved.ID).Scan(&scoreCount); err != nil {
		t.Fatal(err)
	}
	if answerCount != 0 || scoreCount != 0 {
		t.Fatalf("answers = %d, scores = %d, want 0 and 0", answerCount, scoreCount)
	}

	var status string
	if err := testDB.Pool().QueryRow(ctx, "SELECT status FROM effect_outbox WHERE id = $1", effect.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "done" {
		t.Fatalf("effect status = %q, want done (the new fingerprint got its own queued effect)", status)
	}
}

package db_test

import (
	"os"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/pgtest"
)

func TestOwnOptionAnswersMigration(t *testing.T) {
	pool := pgtest.New(t)
	ctx := t.Context()

	raw, err := os.ReadFile("scripts/migrations/20261009172100_own_option_answers_per_user.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, down, _ := strings.Cut(strings.TrimPrefix(string(raw), "-- +goose Up"), "-- +goose Down")
	if _, err := pool.Exec(ctx, down); err != nil {
		t.Fatalf("down: %v", err)
	}

	scoredAtFingerprint := pgtest.InsertUser(t, pool)
	scoredAtOldFingerprint := pgtest.InsertUser(t, pool)
	unscored := pgtest.InsertUser(t, pool)
	jobID := pgtest.InsertJob(t, pool, "Engineer", "fp-1")
	for userID, fingerprint := range map[string]string{scoredAtFingerprint: "fp-1", scoredAtOldFingerprint: "fp-0"} {
		_, err := pool.Exec(ctx,
			`INSERT INTO job_scores (job_id, user_id, suitability_score, breakdown, score_fingerprint) VALUES ($1, $2, 50, '[]', $3)`,
			jobID, userID, fingerprint)
		if err != nil {
			t.Fatalf("insert score: %v", err)
		}
	}
	_, err = pool.Exec(ctx,
		`INSERT INTO option_answers (job_id, fingerprint, question_hash, model, p_yes, p_no, p_not_stated, confidence)
		 VALUES ($1, 'fp-1', 'hash-1', 'm', 0.9, 0.05, 0.05, 0.9)`, jobID)
	if err != nil {
		t.Fatalf("insert legacy answer: %v", err)
	}

	if _, err := pool.Exec(ctx, up); err != nil {
		t.Fatalf("up: %v", err)
	}

	rows, err := pool.Query(ctx, `SELECT user_id FROM option_answers WHERE job_id = $1`, jobID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var owners []string
	for rows.Next() {
		var owner string
		if err := rows.Scan(&owner); err != nil {
			t.Fatal(err)
		}
		owners = append(owners, owner)
	}
	if len(owners) != 1 || owners[0] != scoredAtFingerprint {
		t.Errorf("answer owners = %v, want only %s (scored at fp-1; %s scored at fp-0, %s unscored)",
			owners, scoredAtFingerprint, scoredAtOldFingerprint, unscored)
	}
}

package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestCandidateRetentionAndDuplicateCards(t *testing.T) {
	ctx := context.Background()
	if _, err := testDB.Pool().Exec(ctx, "TRUNCATE job_candidates, source_targets, users CASCADE"); err != nil {
		t.Fatal(err)
	}
	user, err := testDB.CreateUser(ctx, "candidate-user", "hash", "")
	if err != nil {
		t.Fatal(err)
	}
	target, err := testDB.CreateSourceTarget(ctx, user.ID, "wis", "engineer", true, nil)
	if err != nil {
		t.Fatal(err)
	}
	card := dto.Job{URL: "https://example.com/jobs/1#details", Title: "Senior Engineer", CompanySlug: "acme", Location: "London"}
	got, err := testDB.SaveCards(ctx, target, []dto.Job{card, card})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != got[1].ID || got[0].URL != "https://example.com/jobs/1" {
		t.Fatalf("duplicate candidates: %+v", got)
	}
	var expiry time.Time
	if err := testDB.Pool().QueryRow(ctx, "SELECT expires_at FROM job_candidates WHERE id = $1", got[0].ID).Scan(&expiry); err != nil {
		t.Fatal(err)
	}
	if expiry.Before(time.Now().Add(59*24*time.Hour)) || expiry.After(time.Now().Add(61*24*time.Hour)) {
		t.Fatalf("unexpected expiry: %s", expiry)
	}

	if _, err := testDB.Pool().Exec(ctx, "UPDATE job_candidates SET expires_at = NOW() - INTERVAL '1 second'"); err != nil {
		t.Fatal(err)
	}
	retained, err := testDB.ListForUser(ctx, user.ID, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(retained) != 0 {
		t.Fatalf("expired candidate remained queryable: %+v", retained)
	}
	if _, err := testDB.SaveCards(ctx, target, []dto.Job{card}); err != nil {
		t.Fatal(err)
	}
	retained, err = testDB.ListForUser(ctx, user.ID, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(retained) != 1 {
		t.Fatalf("rediscovered candidate missing: %+v", retained)
	}
	version := time.Now().UTC()
	requested, err := testDB.Assess(ctx, got[0].ID, user.ID, version, false)
	if err != nil || requested {
		t.Fatalf("rejected assessment requested detail: requested=%v err=%v", requested, err)
	}
	requested, err = testDB.Assess(ctx, got[0].ID, user.ID, version.Add(time.Second), true)
	if err != nil || !requested {
		t.Fatalf("newly relevant candidate not requested: requested=%v err=%v", requested, err)
	}
	requested, err = testDB.Assess(ctx, got[0].ID, user.ID, version.Add(2*time.Second), true)
	if err != nil || requested {
		t.Fatalf("duplicate detail request: requested=%v err=%v", requested, err)
	}
	var assessments int
	if err := testDB.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM candidate_assessments WHERE candidate_id = $1 AND user_id = $2", got[0].ID, user.ID).Scan(&assessments); err != nil {
		t.Fatal(err)
	}
	if assessments != 1 {
		t.Fatalf("assessment rows = %d, want one latest row", assessments)
	}
}

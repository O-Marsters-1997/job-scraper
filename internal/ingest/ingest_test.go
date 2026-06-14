package ingest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/ingest"
)

type stubSaver struct {
	calls [][]dto.Job
	err   error
}

func (s *stubSaver) Save(_ context.Context, jobs []dto.Job) error {
	if s.err != nil {
		return s.err
	}
	s.calls = append(s.calls, jobs)
	return nil
}

type stubScorer struct {
	jobs []dto.Job
}

func (s *stubScorer) ScoreAndSave(_ context.Context, job dto.Job) {
	s.jobs = append(s.jobs, job)
}

type stubNotifier struct {
	jobs []dto.Job
}

func (n *stubNotifier) NotifyNewJob(_ context.Context, job dto.Job) {
	n.jobs = append(n.jobs, job)
}

func TestIngest_ValidJobs(t *testing.T) {
	db := &stubSaver{}
	scorer := &stubScorer{}
	notifier := &stubNotifier{}
	ing := ingest.New(db, scorer, notifier)

	jobs := []dto.Job{
		{Title: "Engineer", URL: "https://example.com/1"},
		{Title: "Manager", URL: "https://example.com/2"},
	}
	if err := ing.Ingest(context.Background(), jobs); err != nil {
		t.Fatal(err)
	}

	if len(db.calls) != 1 || len(db.calls[0]) != 2 {
		t.Errorf("Save called with %d batches / %v jobs, want 1 batch of 2", len(db.calls), db.calls)
	}
	if len(scorer.jobs) != 2 {
		t.Errorf("scorer called %d times, want 2", len(scorer.jobs))
	}
	if len(notifier.jobs) != 2 {
		t.Errorf("notifier called %d times, want 2", len(notifier.jobs))
	}
}

func TestIngest_FiltersInvalidJobs(t *testing.T) {
	db := &stubSaver{}
	scorer := &stubScorer{}
	ing := ingest.New(db, scorer, nil)

	jobs := []dto.Job{
		{Title: "", URL: "https://example.com/1"}, // missing title
		{Title: "Engineer", URL: ""},              // missing url
		{Title: "Valid", URL: "https://example.com/3"},
	}
	if err := ing.Ingest(context.Background(), jobs); err != nil {
		t.Fatal(err)
	}

	if len(db.calls) != 1 || len(db.calls[0]) != 1 {
		t.Errorf("expected 1 valid job saved, got batches: %v", db.calls)
	}
	if db.calls[0][0].Title != "Valid" {
		t.Errorf("wrong job saved: %q", db.calls[0][0].Title)
	}
	if len(scorer.jobs) != 1 {
		t.Errorf("scorer called %d times, want 1", len(scorer.jobs))
	}
}

func TestIngest_AllInvalid_ReturnsNil(t *testing.T) {
	db := &stubSaver{}
	ing := ingest.New(db, nil, nil)

	if err := ing.Ingest(context.Background(), []dto.Job{{Title: "", URL: ""}}); err != nil {
		t.Errorf("want nil error for all-invalid input, got %v", err)
	}
	if len(db.calls) != 0 {
		t.Errorf("expected no Save calls, got %d", len(db.calls))
	}
}

func TestIngest_SaveError_Propagates(t *testing.T) {
	db := &stubSaver{err: errors.New("db down")}
	scorer := &stubScorer{}
	notifier := &stubNotifier{}
	ing := ingest.New(db, scorer, notifier)

	err := ing.Ingest(context.Background(), []dto.Job{{Title: "Engineer", URL: "https://example.com/1"}})
	if err == nil {
		t.Error("expected error from Save, got nil")
	}
	if len(scorer.jobs) != 0 {
		t.Error("scorer must not be called after Save error")
	}
	if len(notifier.jobs) != 0 {
		t.Error("notifier must not be called after Save error")
	}
}

func TestIngest_NilScorerAndNotifier_NoPanic(t *testing.T) {
	db := &stubSaver{}
	ing := ingest.New(db, nil, nil)

	if err := ing.Ingest(context.Background(), []dto.Job{{Title: "Engineer", URL: "https://example.com/1"}}); err != nil {
		t.Errorf("unexpected error with nil scorer/notifier: %v", err)
	}
	if len(db.calls) != 1 {
		t.Errorf("expected Save called once, got %d", len(db.calls))
	}
}

func TestIngest_EmptySlice(t *testing.T) {
	db := &stubSaver{}
	ing := ingest.New(db, nil, nil)

	if err := ing.Ingest(context.Background(), nil); err != nil {
		t.Errorf("unexpected error for empty input: %v", err)
	}
	if len(db.calls) != 0 {
		t.Errorf("expected no Save calls for empty input, got %d", len(db.calls))
	}
}

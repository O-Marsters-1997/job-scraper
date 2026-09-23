package score_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/score"
)

type effectStore struct {
	effect    dto.ScoringEffect
	job       dto.Job
	failed    string
	completed bool
}

func (s *effectStore) ClaimScoringEffect(context.Context) (dto.ScoringEffect, error) {
	return s.effect, nil
}
func (s *effectStore) GetJob(context.Context, string, string) (dto.Job, error) { return s.job, nil }
func (s *effectStore) GetSearchConfig(context.Context, string) (dto.SearchConfig, error) {
	return dto.SearchConfig{}, nil
}
func (s *effectStore) FailScoringEffect(_ context.Context, _ string, _ int, reason string) error {
	s.failed = reason
	return nil
}
func (s *effectStore) CompleteScoringEffect(context.Context, dto.ScoringEffect, int, string, []string, []string) error {
	s.completed = true
	return nil
}

type failingScorer struct{}

func (failingScorer) Score(context.Context, dto.Job, dto.SearchConfig, string) (score.SuitabilityResult, error) {
	return score.SuitabilityResult{}, errors.New("AI unavailable")
}
func (failingScorer) ScoreBatch(context.Context, []dto.Job, dto.SearchConfig, string) ([]score.SuitabilityResult, error) {
	return nil, errors.New("AI unavailable")
}

func TestOutboxWorker_RetriesScoringFailure(t *testing.T) {
	store := &effectStore{
		effect: dto.ScoringEffect{ID: "effect", JobID: "job", UserID: "user", Fingerprint: "same", Model: score.DefaultSuitabilityModel, Attempts: 1},
		job:    dto.Job{ID: "job", ContentFingerprint: "same"},
	}
	worker := score.NewOutboxWorker(store, func(context.Context, string) (string, error) { return "key", nil }, func(string) score.SuitabilityScorer { return failingScorer{} })
	if err := worker.RunOnce(context.Background()); err == nil {
		t.Fatal("expected scoring error")
	}
	if store.failed == "" || store.completed {
		t.Fatalf("failure handling = failed %q completed %v", store.failed, store.completed)
	}
}

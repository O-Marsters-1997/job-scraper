package score_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/score"
)

type effectStore struct {
	effect      dto.ScoringEffect
	job         dto.Job
	config      dto.SearchConfig
	profile     dto.Profile
	failed      string
	completed   bool
	completeErr error
	unsaved     bool
}

func (s *effectStore) ClaimScoringEffect(context.Context) (dto.ScoringEffect, error) {
	return s.effect, nil
}
func (s *effectStore) GetJob(context.Context, string, string) (dto.Job, error) { return s.job, nil }
func (s *effectStore) GetSearchConfig(context.Context, string) (dto.SearchConfig, error) {
	return s.config, nil
}
func (s *effectStore) GetProfile(context.Context, string) (dto.Profile, error) { return s.profile, nil }
func (s *effectStore) FailScoringEffect(_ context.Context, _ string, _ int, reason string) error {
	s.failed = reason
	return nil
}
func (s *effectStore) CompleteScoringEffect(context.Context, dto.ScoringEffect, int, string, []string, []string) (bool, error) {
	if s.completeErr != nil {
		return false, s.completeErr
	}
	s.completed = true
	return !s.unsaved, nil
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
		effect: dto.ScoringEffect{ID: "effect", JobID: "job", UserID: "user", Fingerprint: "same", Model: score.DefaultSuitabilityModel, Attempts: 1, FirstDiscovery: true},
		job:    dto.Job{ID: "job", ContentFingerprint: "same"},
	}
	sent := false
	worker := score.NewOutboxWorker(store, func(context.Context, string) (string, error) { return "key", nil }, func(string) score.SuitabilityScorer { return failingScorer{} }, func(context.Context, dto.Job, string) error { sent = true; return nil })
	if err := worker.RunOnce(context.Background()); err == nil {
		t.Fatal("expected scoring error")
	}
	if store.failed == "" || store.completed || sent {
		t.Fatalf("failure handling = failed %q completed %v sent %v", store.failed, store.completed, sent)
	}
}

type fixedScorer struct{ value int }

func (s fixedScorer) Score(context.Context, dto.Job, dto.SearchConfig, string) (score.SuitabilityResult, error) {
	return score.SuitabilityResult{Score: s.value}, nil
}
func (s fixedScorer) ScoreBatch(context.Context, []dto.Job, dto.SearchConfig, string) ([]score.SuitabilityResult, error) {
	return nil, nil
}

func TestOutboxWorkerRoutesFirstDiscoveryPerUser(t *testing.T) {
	tests := []struct {
		name, email              string
		threshold, score         int
		firstDiscovery, wantSent bool
	}{
		{"alice qualifies", "alice@example.com", 80, 85, true, true},
		{"bob qualifies at own threshold", "bob@example.com", 90, 95, true, true},
		{"bob below own threshold", "bob@example.com", 90, 85, true, false},
		{"missing email", "", 80, 95, true, false},
		{"later assessment", "alice@example.com", 80, 95, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &effectStore{effect: dto.ScoringEffect{ID: "effect", JobID: "job", UserID: tt.name, Fingerprint: "same", Model: score.DefaultSuitabilityModel, FirstDiscovery: tt.firstDiscovery}, job: dto.Job{ID: "job", ContentFingerprint: "same"}, config: dto.SearchConfig{NotifyThreshold: tt.threshold}, profile: dto.Profile{Email: tt.email}}
			var recipients []string
			worker := score.NewOutboxWorker(store, func(context.Context, string) (string, error) { return "key", nil }, func(string) score.SuitabilityScorer { return fixedScorer{tt.score} }, func(_ context.Context, _ dto.Job, email string) error {
				recipients = append(recipients, email)
				return nil
			})
			if err := worker.RunOnce(t.Context()); err != nil {
				t.Fatal(err)
			}
			if !store.completed {
				t.Fatal("score not completed")
			}
			if (len(recipients) == 1) != tt.wantSent {
				t.Fatalf("recipients: %v", recipients)
			}
			if tt.wantSent && recipients[0] != tt.email {
				t.Fatalf("recipient %q, want %q", recipients[0], tt.email)
			}
		})
	}
}

func TestOutboxWorkerDoesNotSendUntilScoreIsStored(t *testing.T) {
	store := &effectStore{effect: dto.ScoringEffect{ID: "effect", JobID: "job", UserID: "alice", Fingerprint: "same", Model: score.DefaultSuitabilityModel, FirstDiscovery: true}, job: dto.Job{ID: "job", ContentFingerprint: "same"}, profile: dto.Profile{Email: "alice@example.com"}, completeErr: errors.New("database unavailable")}
	sent := false
	worker := score.NewOutboxWorker(store, func(context.Context, string) (string, error) { return "key", nil }, func(string) score.SuitabilityScorer { return fixedScorer{95} }, func(context.Context, dto.Job, string) error { sent = true; return nil })
	if err := worker.RunOnce(t.Context()); err == nil || sent {
		t.Fatalf("complete failure: err=%v sent=%v", err, sent)
	}
}

func TestOutboxWorkerDoesNotSendWhenCompletionIsStale(t *testing.T) {
	store := &effectStore{effect: dto.ScoringEffect{ID: "effect", JobID: "job", UserID: "alice", Fingerprint: "same", Model: score.DefaultSuitabilityModel, FirstDiscovery: true}, job: dto.Job{ID: "job", ContentFingerprint: "same"}, profile: dto.Profile{Email: "alice@example.com"}, unsaved: true}
	sent := false
	worker := score.NewOutboxWorker(store, func(context.Context, string) (string, error) { return "key", nil }, func(string) score.SuitabilityScorer { return fixedScorer{95} }, func(context.Context, dto.Job, string) error { sent = true; return nil })
	if err := worker.RunOnce(t.Context()); err != nil || sent {
		t.Fatalf("stale completion: err=%v sent=%v", err, sent)
	}
}

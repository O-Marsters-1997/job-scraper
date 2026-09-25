package score_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

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

type queueStore struct {
	mu      sync.Mutex
	effects []dto.ScoringEffect
	jobs    map[string]dto.Job
	config  dto.SearchConfig
	profile dto.Profile

	resultMu  sync.Mutex
	completed []string
	failed    []string
}

func (s *queueStore) ClaimScoringEffect(context.Context) (dto.ScoringEffect, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.effects) == 0 {
		return dto.ScoringEffect{}, pgx.ErrNoRows
	}
	effect := s.effects[0]
	s.effects = s.effects[1:]
	return effect, nil
}

func (s *queueStore) GetJob(_ context.Context, jobID, _ string) (dto.Job, error) {
	return s.jobs[jobID], nil
}
func (s *queueStore) GetSearchConfig(context.Context, string) (dto.SearchConfig, error) {
	return s.config, nil
}
func (s *queueStore) GetProfile(context.Context, string) (dto.Profile, error) { return s.profile, nil }

func (s *queueStore) FailScoringEffect(_ context.Context, id string, _ int, _ string) error {
	s.resultMu.Lock()
	defer s.resultMu.Unlock()
	s.failed = append(s.failed, id)
	return nil
}

func (s *queueStore) CompleteScoringEffect(_ context.Context, effect dto.ScoringEffect, _ int, _ string, _, _ []string) (bool, error) {
	s.resultMu.Lock()
	defer s.resultMu.Unlock()
	s.completed = append(s.completed, effect.ID)
	return true, nil
}

func newQueueStore(n int) (*queueStore, []dto.ScoringEffect) {
	effects := make([]dto.ScoringEffect, n)
	jobs := make(map[string]dto.Job, n)
	for i := range effects {
		jobID := fmt.Sprintf("job-%d", i)
		effects[i] = dto.ScoringEffect{ID: fmt.Sprintf("effect-%d", i), JobID: jobID, UserID: "user", Fingerprint: "same", Model: score.DefaultSuitabilityModel}
		jobs[jobID] = dto.Job{ID: jobID, ContentFingerprint: "same"}
	}
	return &queueStore{effects: effects, jobs: jobs}, effects
}

func TestOutboxWorkerRunTickDrainsQueue(t *testing.T) {
	store, effects := newQueueStore(10)
	worker := score.NewOutboxWorker(store, func(context.Context, string) (string, error) { return "key", nil }, func(string) score.SuitabilityScorer { return fixedScorer{50} }, nil)

	if err := worker.RunTick(t.Context()); err != nil {
		t.Fatalf("RunTick: %v", err)
	}
	if len(store.completed) != len(effects) {
		t.Fatalf("completed %d effects, want %d", len(store.completed), len(effects))
	}
}

type concurrencyScorer struct {
	mu       sync.Mutex
	inFlight int
	maxSeen  int
}

func (s *concurrencyScorer) Score(context.Context, dto.Job, dto.SearchConfig, string) (score.SuitabilityResult, error) {
	s.mu.Lock()
	s.inFlight++
	if s.inFlight > s.maxSeen {
		s.maxSeen = s.inFlight
	}
	s.mu.Unlock()

	time.Sleep(10 * time.Millisecond)

	s.mu.Lock()
	s.inFlight--
	s.mu.Unlock()
	return score.SuitabilityResult{Score: 50}, nil
}

func TestOutboxWorkerRunTickLimitsConcurrency(t *testing.T) {
	store, _ := newQueueStore(20)
	scorer := &concurrencyScorer{}
	worker := score.NewOutboxWorker(store, func(context.Context, string) (string, error) { return "key", nil }, func(string) score.SuitabilityScorer { return scorer }, nil)

	if err := worker.RunTick(t.Context()); err != nil {
		t.Fatalf("RunTick: %v", err)
	}
	if scorer.maxSeen > 4 {
		t.Fatalf("max concurrent scorer calls = %d, want <= 4", scorer.maxSeen)
	}
	if scorer.maxSeen < 2 {
		t.Fatalf("expected overlapping scorer calls, max concurrent = %d", scorer.maxSeen)
	}
}

type selectiveFailScorer struct{ failJobID string }

func (s selectiveFailScorer) Score(_ context.Context, job dto.Job, _ dto.SearchConfig, _ string) (score.SuitabilityResult, error) {
	if job.ID == s.failJobID {
		return score.SuitabilityResult{}, errors.New("boom")
	}
	return score.SuitabilityResult{Score: 50}, nil
}

func TestOutboxWorkerRunTickContinuesAfterEffectFailure(t *testing.T) {
	store, effects := newQueueStore(5)
	scorer := selectiveFailScorer{failJobID: "job-2"}
	worker := score.NewOutboxWorker(store, func(context.Context, string) (string, error) { return "key", nil }, func(string) score.SuitabilityScorer { return scorer }, nil)

	if err := worker.RunTick(t.Context()); err != nil {
		t.Fatalf("RunTick: %v", err)
	}
	if len(store.failed) != 1 || store.failed[0] != "effect-2" {
		t.Fatalf("failed = %v, want [effect-2]", store.failed)
	}
	if len(store.completed) != len(effects)-1 {
		t.Fatalf("completed = %d, want %d", len(store.completed), len(effects)-1)
	}
}

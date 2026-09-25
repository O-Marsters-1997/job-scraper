package jobreasoning

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/credstore"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/score"
)

type fakeCredentialStore struct {
	key string
	err error
}

func (f fakeCredentialStore) Get(_ context.Context, _, _ string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.key, nil
}

type fakeScorer struct {
	result score.SuitabilityResult
	err    error
	calls  *int
}

func (f fakeScorer) Score(_ context.Context, _ dto.Job, _ dto.SearchConfig, _ string) (score.SuitabilityResult, error) {
	if f.calls != nil {
		*f.calls++
	}
	return f.result, f.err
}

func newTestService(t *testing.T) (*Service, *providers.MockJobScoreProvider, *providers.MockJobProvider, *int) {
	t.Helper()
	scores := providers.NewMockJobScoreProvider()
	jobs := providers.NewMockJobProvider()
	configs := providers.NewMockSearchConfigProvider()
	prefs := providers.NewMockUserAIPrefsProvider()
	creds := fakeCredentialStore{key: "sk-test"}
	calls := new(int)
	scorer := fakeScorer{result: score.SuitabilityResult{Score: 80, Rationale: "good fit"}, calls: calls}
	svc := New(scores, jobs, configs, prefs, creds, func(string) score.SuitabilityScorer { return scorer })
	return svc, scores, jobs, calls
}

func TestGenerate_JobNotYetScored(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	_, err := svc.Generate(context.Background(), "user-1", "job-1")
	if status, ok := apperr.StatusFor(err); !ok || status != http.StatusUnprocessableEntity {
		t.Fatalf("status = %v, ok = %v, want %d", status, ok, http.StatusUnprocessableEntity)
	}
}

func TestGenerate_SkippedOrPendingScoreIsUnprocessable(t *testing.T) {
	tests := []struct {
		name  string
		score dto.JobScore
	}{
		{name: "skipped", score: dto.JobScore{JobID: "job-1", SuitabilitySkipped: true}},
		{name: "pending", score: dto.JobScore{JobID: "job-1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, scores, _, _ := newTestService(t)
			scores.SetJobScore(tt.score)
			_, err := svc.Generate(context.Background(), "user-1", "job-1")
			if status, ok := apperr.StatusFor(err); !ok || status != http.StatusUnprocessableEntity {
				t.Fatalf("status = %v, ok = %v, want %d", status, ok, http.StatusUnprocessableEntity)
			}
		})
	}
}

func TestGenerate_ReturnsCachedReasoningWithoutScoring(t *testing.T) {
	svc, scores, _, calls := newTestService(t)
	suitability := 70
	reasoning := "already reasoned"
	scores.SetJobScore(dto.JobScore{JobID: "job-1", SuitabilityScore: &suitability, Reasoning: &reasoning})

	got, err := svc.Generate(context.Background(), "user-1", "job-1")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got.Reasoning == nil || *got.Reasoning != reasoning {
		t.Fatalf("Reasoning = %v, want %q", got.Reasoning, reasoning)
	}
	if *calls != 0 {
		t.Fatalf("scorer called %d times, want 0", *calls)
	}
}

func TestGenerate_JobNotFound(t *testing.T) {
	svc, scores, _, _ := newTestService(t)
	suitability := 70
	scores.SetJobScore(dto.JobScore{JobID: "job-1", SuitabilityScore: &suitability})

	_, err := svc.Generate(context.Background(), "user-1", "job-1")
	if status, ok := apperr.StatusFor(err); !ok || status != http.StatusNotFound {
		t.Fatalf("status = %v, ok = %v, want %d", status, ok, http.StatusNotFound)
	}
}

func TestGenerate_CredentialMissing(t *testing.T) {
	scores := providers.NewMockJobScoreProvider()
	jobs := providers.NewMockJobProvider()
	configs := providers.NewMockSearchConfigProvider()
	prefs := providers.NewMockUserAIPrefsProvider()
	creds := fakeCredentialStore{err: credstore.ErrNotFound}
	scorer := fakeScorer{result: score.SuitabilityResult{Score: 80}}
	svc := New(scores, jobs, configs, prefs, creds, func(string) score.SuitabilityScorer { return scorer })

	saved, err := jobs.Save(context.Background(), []dto.Job{{URL: "https://example.com/job-1"}})
	if err != nil {
		t.Fatal(err)
	}
	suitability := 70
	scores.SetJobScore(dto.JobScore{JobID: saved[0].ID, SuitabilityScore: &suitability})

	_, err = svc.Generate(context.Background(), "user-1", saved[0].ID)
	if status, ok := apperr.StatusFor(err); !ok || status != http.StatusUnprocessableEntity {
		t.Fatalf("status = %v, ok = %v, want %d", status, ok, http.StatusUnprocessableEntity)
	}
}

func TestGenerate_ScorerFailureIsUpstream(t *testing.T) {
	scores := providers.NewMockJobScoreProvider()
	jobs := providers.NewMockJobProvider()
	configs := providers.NewMockSearchConfigProvider()
	prefs := providers.NewMockUserAIPrefsProvider()
	creds := fakeCredentialStore{key: "sk-test"}
	scorer := fakeScorer{err: errors.New("upstream boom")}
	svc := New(scores, jobs, configs, prefs, creds, func(string) score.SuitabilityScorer { return scorer })

	saved, err := jobs.Save(context.Background(), []dto.Job{{URL: "https://example.com/job-1"}})
	if err != nil {
		t.Fatal(err)
	}
	suitability := 70
	scores.SetJobScore(dto.JobScore{JobID: saved[0].ID, SuitabilityScore: &suitability})

	_, err = svc.Generate(context.Background(), "user-1", saved[0].ID)
	if status, ok := apperr.StatusFor(err); !ok || status != http.StatusBadGateway {
		t.Fatalf("status = %v, ok = %v, want %d", status, ok, http.StatusBadGateway)
	}
}

func TestGenerate_HappyPathUsesPreferredModelAndPersists(t *testing.T) {
	scores := providers.NewMockJobScoreProvider()
	jobs := providers.NewMockJobProvider()
	configs := providers.NewMockSearchConfigProvider()
	prefs := providers.NewMockUserAIPrefsProvider()
	creds := fakeCredentialStore{key: "sk-test"}
	scorer := fakeScorer{result: score.SuitabilityResult{Score: 91, Rationale: "great fit", Matched: []string{"go"}, Missing: []string{"rust"}}}
	svc := New(scores, jobs, configs, prefs, creds, func(string) score.SuitabilityScorer { return scorer })

	if _, err := prefs.UpsertUserAIPrefs(context.Background(), "user-1", "", "claude-opus"); err != nil {
		t.Fatal(err)
	}
	saved, err := jobs.Save(context.Background(), []dto.Job{{URL: "https://example.com/job-1"}})
	if err != nil {
		t.Fatal(err)
	}
	suitability := 40
	scores.SetJobScore(dto.JobScore{JobID: saved[0].ID, SuitabilityScore: &suitability})

	got, err := svc.Generate(context.Background(), "user-1", saved[0].ID)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got.SuitabilityScore == nil || *got.SuitabilityScore != 91 {
		t.Fatalf("SuitabilityScore = %v, want 91", got.SuitabilityScore)
	}
	if got.Reasoning == nil || *got.Reasoning != "great fit" {
		t.Fatalf("Reasoning = %v, want %q", got.Reasoning, "great fit")
	}
}

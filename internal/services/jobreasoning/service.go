// Package jobreasoning generates on-demand suitability reasoning for an
// already-scored job: it re-scores the job with the user's configured
// reasoning model, overwriting the provisional ingest-time score with the
// authoritative result. If reasoning already exists, the cached result is
// returned without a new scoring call.
package jobreasoning

import (
	"context"
	"errors"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/credstore"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/ingest"
	"github.com/ollymarsters/job-scraper/internal/score"
)

type credentialStore interface {
	Get(ctx context.Context, userID, provider string) (string, error)
}

type Service struct {
	scores    providers.JobScoreProvider
	jobs      providers.JobProvider
	configs   providers.SearchConfigProvider
	prefs     providers.UserAIPrefsProvider
	creds     credentialStore
	newScorer func(apiKey string) score.SuitabilityScorer
}

func New(
	scores providers.JobScoreProvider,
	jobs providers.JobProvider,
	configs providers.SearchConfigProvider,
	prefs providers.UserAIPrefsProvider,
	creds credentialStore,
	newScorer func(apiKey string) score.SuitabilityScorer,
) *Service {
	return &Service{
		scores:    scores,
		jobs:      jobs,
		configs:   configs,
		prefs:     prefs,
		creds:     creds,
		newScorer: newScorer,
	}
}

// Generate returns the reasoning for jobID, generating and persisting it if
// it doesn't already exist.
func (s *Service) Generate(ctx context.Context, userID, jobID string) (dto.JobScore, error) {
	existing, err := s.scores.GetJobScore(ctx, jobID, userID)
	if err != nil {
		if errors.Is(err, providers.ErrNotFound) {
			return dto.JobScore{}, apperr.Unprocessable("job not yet scored")
		}
		return dto.JobScore{}, err
	}
	if existing.SuitabilitySkipped || existing.SuitabilityScore == nil {
		return dto.JobScore{}, apperr.Unprocessable("job not scored — skipped or pending")
	}
	if existing.Reasoning != nil {
		return existing, nil
	}

	job, err := s.jobs.GetJob(ctx, jobID, userID)
	if err != nil {
		if errors.Is(err, providers.ErrNotFound) {
			return dto.JobScore{}, apperr.NotFound("job not found")
		}
		return dto.JobScore{}, err
	}

	cfg, err := s.configs.GetSearchConfig(ctx, userID)
	if err != nil && !errors.Is(err, providers.ErrNotFound) {
		return dto.JobScore{}, err
	}

	reasoningModel := score.DefaultReasoningModel
	prefs, err := s.prefs.GetUserAIPrefs(ctx, userID)
	if err != nil && !errors.Is(err, providers.ErrNotFound) {
		return dto.JobScore{}, err
	}
	if err == nil && prefs.ReasoningModel != "" {
		reasoningModel = prefs.ReasoningModel
	}

	provider := ingest.ProviderForModel(reasoningModel)
	apiKey, err := s.creds.Get(ctx, userID, provider)
	if err != nil {
		if errors.Is(err, credstore.ErrNotFound) {
			return dto.JobScore{}, apperr.Unprocessable("scoring not enabled — configure an API key")
		}
		return dto.JobScore{}, err
	}

	result, err := s.newScorer(apiKey).Score(ctx, job, cfg, reasoningModel)
	if err != nil {
		return dto.JobScore{}, apperr.Upstream("scoring failed")
	}

	if err := s.scores.UpsertJobScoreSuitability(ctx, jobID, userID, result.Score, result.Rationale, result.Matched, result.Missing); err != nil {
		return dto.JobScore{}, err
	}

	return s.scores.GetJobScore(ctx, jobID, userID)
}

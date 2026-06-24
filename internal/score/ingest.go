package score

import (
	"context"
	"errors"
	"log/slog"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

const defaultSuitabilityModel = "claude-haiku-4-5-20251001"

type ScoreWriter interface {
	UpsertJobScoreSuitability(ctx context.Context, jobID, userID string, score int, reasoning string, matched, missing []string) error
}

type ConfigReader interface {
	GetSearchConfig(ctx context.Context, userID string) (dto.SearchConfig, error)
}

type UserAIPrefsReader interface {
	GetUserAIPrefs(ctx context.Context, userID string) (dto.UserAIPrefs, error)
}

type IngestScorer struct {
	scorer    SuitabilityScorer
	db        ScoreWriter
	cfgDB     ConfigReader
	aiPrefsDB UserAIPrefsReader
	userID    string
}

func NewIngestScorer(scorer SuitabilityScorer, db ScoreWriter, cfgDB ConfigReader, userID string) *IngestScorer {
	return &IngestScorer{scorer: scorer, db: db, cfgDB: cfgDB, userID: userID}
}

// aiPrefsDB may be nil; the default model is used when it is.
func NewIngestScorerWithPrefs(scorer SuitabilityScorer, db ScoreWriter, cfgDB ConfigReader, aiPrefsDB UserAIPrefsReader, userID string) *IngestScorer {
	return &IngestScorer{scorer: scorer, db: db, cfgDB: cfgDB, aiPrefsDB: aiPrefsDB, userID: userID}
}

// ScoreAndSave scores the job, persists the result, and returns the score (0 on any error).
// Errors are logged internally so a scoring failure never blocks the ingest path.
func (s *IngestScorer) ScoreAndSave(ctx context.Context, job dto.Job) int {
	cfg, err := s.cfgDB.GetSearchConfig(ctx, s.userID)
	if err != nil {
		slog.Warn("suitability: could not load search config", slog.Any("err", err))
		return 0
	}

	modelID := defaultSuitabilityModel
	if s.aiPrefsDB != nil {
		prefs, err := s.aiPrefsDB.GetUserAIPrefs(ctx, s.userID)
		if err != nil && !errors.Is(err, providers.ErrNotFound) {
			slog.Warn("suitability: could not load ai prefs, using default model", slog.Any("err", err))
		} else if err == nil {
			modelID = prefs.SuitabilityModel
		}
	}

	result, err := s.scorer.Score(ctx, job, cfg, modelID)
	if err != nil {
		slog.Error("suitability score failed", slog.String("url", job.URL), slog.Any("err", err))
		return 0
	}

	slog.Info("suitability scored",
		slog.String("url", job.URL),
		slog.Int("score", result.Score),
		slog.Int("input_tokens", result.Usage.InputTokens),
		slog.Int("output_tokens", result.Usage.OutputTokens),
		slog.Float64("cost_usd", result.Usage.CostUSD),
	)

	if err := s.db.UpsertJobScoreSuitability(ctx, job.ID, s.userID, result.Score, result.Rationale, result.Matched, result.Missing); err != nil {
		slog.Error("upsert suitability failed", slog.String("url", job.URL), slog.Any("err", err))
	}
	return result.Score
}

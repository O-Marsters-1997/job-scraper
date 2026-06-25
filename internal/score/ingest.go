package score

import (
	"context"
	"errors"
	"log/slog"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

// DefaultSuitabilityModel is the model used for suitability scoring when the
// user has no AI prefs or the prefs do not specify a model.
const DefaultSuitabilityModel = "claude-haiku-4-5-20251001"

type ScoreWriter interface {
	UpsertJobScoreSuitability(ctx context.Context, jobID, userID string, score int, reasoning string, matched, missing []string) error
	UpsertJobScoreSkipped(ctx context.Context, jobID, userID string) error
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
}

func NewIngestScorer(scorer SuitabilityScorer, db ScoreWriter, cfgDB ConfigReader) *IngestScorer {
	return &IngestScorer{scorer: scorer, db: db, cfgDB: cfgDB}
}

// aiPrefsDB may be nil; the default model is used when it is.
func NewIngestScorerWithPrefs(scorer SuitabilityScorer, db ScoreWriter, cfgDB ConfigReader, aiPrefsDB UserAIPrefsReader) *IngestScorer {
	return &IngestScorer{scorer: scorer, db: db, cfgDB: cfgDB, aiPrefsDB: aiPrefsDB}
}

// ScoreAndSave scores the job for userID, persists the result, and returns the score (0 on any error).
// Errors are logged internally so a scoring failure never blocks the ingest path.
func (s *IngestScorer) ScoreAndSave(ctx context.Context, job dto.Job, userID string) int {
	cfg, err := s.cfgDB.GetSearchConfig(ctx, userID)
	if err != nil {
		slog.Warn("suitability: could not load search config", slog.Any("err", err))
		return 0
	}

	if cfg.RelevanceCutoff > 0 && job.RelevanceScore != nil && *job.RelevanceScore < cfg.RelevanceCutoff {
		slog.Info("suitability: skipped (below relevance cutoff)",
			slog.String("url", job.URL),
			slog.Int("relevance", *job.RelevanceScore),
			slog.Int("cutoff", cfg.RelevanceCutoff),
		)
		if err := s.db.UpsertJobScoreSkipped(ctx, job.ID, userID); err != nil {
			slog.Error("upsert skipped failed", slog.String("url", job.URL), slog.Any("err", err))
		}
		return 0
	}

	modelID := DefaultSuitabilityModel
	if s.aiPrefsDB != nil {
		prefs, err := s.aiPrefsDB.GetUserAIPrefs(ctx, userID)
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

	if err := s.db.UpsertJobScoreSuitability(ctx, job.ID, userID, result.Score, result.Rationale, result.Matched, result.Missing); err != nil {
		slog.Error("upsert suitability failed", slog.String("url", job.URL), slog.Any("err", err))
	}
	return result.Score
}

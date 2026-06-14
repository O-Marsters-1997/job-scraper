package score

import (
	"context"
	"log/slog"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type ScoreWriter interface {
	UpsertJobScoreSuitability(ctx context.Context, jobID, userID string, score int) error
}

type ConfigReader interface {
	GetSearchConfig(ctx context.Context, userID string) (dto.SearchConfig, error)
}

type IngestScorer struct {
	scorer SuitabilityScorer
	db     ScoreWriter
	cfgDB  ConfigReader
	userID string
}

func NewIngestScorer(scorer SuitabilityScorer, db ScoreWriter, cfgDB ConfigReader, userID string) *IngestScorer {
	return &IngestScorer{scorer: scorer, db: db, cfgDB: cfgDB, userID: userID}
}

// ScoreAndSave scores the job, persists the result, and returns the score (0 on any error).
// Errors are logged internally so a scoring failure never blocks the ingest path.
func (s *IngestScorer) ScoreAndSave(ctx context.Context, job dto.Job) int {
	cfg, err := s.cfgDB.GetSearchConfig(ctx, s.userID)
	if err != nil {
		slog.Warn("suitability: could not load search config", slog.Any("err", err))
		return 0
	}

	sc, usage, err := s.scorer.Score(ctx, job, cfg)
	if err != nil {
		slog.Error("suitability score failed", slog.String("url", job.URL), slog.Any("err", err))
		return 0
	}

	slog.Info("suitability scored",
		slog.String("url", job.URL),
		slog.Int("score", sc),
		slog.Int("input_tokens", usage.InputTokens),
		slog.Int("output_tokens", usage.OutputTokens),
		slog.Float64("cost_usd", usage.CostUSD),
	)

	if err := s.db.UpsertJobScoreSuitability(ctx, job.ID, s.userID, sc); err != nil {
		slog.Error("upsert suitability failed", slog.String("url", job.URL), slog.Any("err", err))
	}
	return sc
}

package scoringconfig

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// Extractor turns a user's free preference text into stances against the
// live option bank, billed to apiKey. It may return a pick for an id or
// stance the bank doesn't have; the caller drops those.
type Extractor interface {
	Extract(ctx context.Context, apiKey, text string, options []dto.ScoringOption, dimensions []dto.DimensionSpec) ([]dto.Pick, error)
}

// Credentials reads a user's stored provider API key.
type Credentials interface {
	Get(ctx context.Context, userID, provider string) (string, error)
}

// Reconsiderer re-evaluates a user's discovered-job candidates against an
// updated Search Config.
type Reconsiderer interface {
	Reconsider(ctx context.Context, config dto.SearchConfig) error
}

// Recomputer re-scores a user's already-scored jobs from cached answers.
type Recomputer interface {
	Recompute(ctx context.Context, userID string) (dto.RecomputeResult, error)
}

// Backfiller queues an answer effect for a user's already-scored jobs
// missing an answer to a newly picked question.
type Backfiller interface {
	FillMissingAnswers(ctx context.Context, userID string) (int64, error)
}

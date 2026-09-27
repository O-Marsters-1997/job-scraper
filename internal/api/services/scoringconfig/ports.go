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

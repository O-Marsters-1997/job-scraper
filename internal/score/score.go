package score

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type SuitabilityResult struct {
	Score     int
	Matched   []string
	Missing   []string
	Rationale string
	Usage     TokenUsage
}

// SuitabilityScorer scores job descriptions against the user's rubric.
// modelID selects which Claude model to use; if empty, the implementation uses its default.
type SuitabilityScorer interface {
	Score(ctx context.Context, job dto.Job, cfg dto.SearchConfig, modelID string) (SuitabilityResult, error)
}

type TokenUsage struct {
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
	CostUSD             float64
}

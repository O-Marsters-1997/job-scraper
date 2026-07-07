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
	// Score returns the full bundle: score, matched, missing, and rationale. Used for on-demand reasoning.
	Score(ctx context.Context, job dto.Job, cfg dto.SearchConfig, modelID string) (SuitabilityResult, error)
	// ScoreBatch scores multiple jobs in fewer API calls. Results are in the same order as jobs.
	ScoreBatch(ctx context.Context, jobs []dto.Job, cfg dto.SearchConfig, modelID string) ([]SuitabilityResult, error)
}

type TokenUsage struct {
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
	CostUSD             float64
}

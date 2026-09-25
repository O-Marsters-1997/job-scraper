package score

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// SuitabilityResult is a scorer's verdict on one job for one user.
type SuitabilityResult struct {
	Score      int
	Criteria   map[string]float64
	Confidence float64
	Model      string
	Cost       float64
}

// SuitabilityScorer scores a job against the user's scoring questions.
type SuitabilityScorer interface {
	Score(ctx context.Context, job dto.Job, cfg dto.SearchConfig, modelID string) (SuitabilityResult, error)
}

package providers

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// ScoringOptionsProvider is the single access point for the scoring option bank.
type ScoringOptionsProvider interface {
	ListScoringOptions(ctx context.Context) ([]dto.ScoringOption, error)
}

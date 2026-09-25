package providers

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type ScoringEffectsProvider interface {
	GetScoringStatus(ctx context.Context, userID string) (dto.ScoringStatus, error)
	QueueRescore(ctx context.Context, userID string) (int64, error)
}

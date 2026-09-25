package providers

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// ScoringEffectsProvider is the single access point for the scoring effect
// outbox: pending/failed/stale counts and queuing a rescore.
type ScoringEffectsProvider interface {
	GetScoringStatus(ctx context.Context, userID string) (dto.ScoringStatus, error)
	QueueRescore(ctx context.Context, userID string) (int64, error)
}

package db

import (
	"context"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
)

var _ telemetry.StateReader = (*DB)(nil)

func (db *DB) OpsState(ctx context.Context) (dto.OpsState, error) {
	row, err := db.queries.OpsState(ctx)
	if err != nil {
		return dto.OpsState{}, fmt.Errorf("ops state: %w", err)
	}
	var oldestPendingAge time.Duration
	if row.OldestPendingCreatedAt.Valid {
		oldestPendingAge = time.Since(row.OldestPendingCreatedAt.Time)
	}
	return dto.OpsState{
		OutboxPending:          row.OutboxPending,
		OutboxOldestPendingAge: oldestPendingAge,
		OutboxFailed:           row.OutboxFailed,
	}, nil
}

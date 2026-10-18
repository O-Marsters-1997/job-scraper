package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store/sqlc"
)

func (s *Store) RecordDiscoverMiss(ctx context.Context, source, token string) error {
	if err := s.queries.RecordDiscoverMiss(ctx, sqlc.RecordDiscoverMissParams{Source: source, BoardToken: token}); err != nil {
		return fmt.Errorf("store.RecordDiscoverMiss: %w", err)
	}
	return nil
}

func (s *Store) DiscoverMissedSince(ctx context.Context, source, token string, since time.Time) (bool, error) {
	missed, err := s.queries.DiscoverMissedSince(ctx, sqlc.DiscoverMissedSinceParams{Source: source, BoardToken: token, MissedAt: pgtype.Timestamptz{Time: since, Valid: true}})
	if err != nil {
		return false, fmt.Errorf("store.DiscoverMissedSince: %w", err)
	}
	return missed, nil
}

package store

import (
	"context"
	"fmt"

	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store/sqlc"
)

func (s *Store) HarvestSeenIDs(ctx context.Context, source string) ([]string, error) {
	ids, err := s.queries.ListHarvestSeenIDs(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("store.HarvestSeenIDs: %w", err)
	}
	return ids, nil
}

func (s *Store) RecordHarvestSeenIDs(ctx context.Context, source, outcome string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if err := s.queries.RecordHarvestSeenIDs(ctx, sqlc.RecordHarvestSeenIDsParams{Source: source, ExternalIds: ids, Outcome: outcome}); err != nil {
		return fmt.Errorf("store.RecordHarvestSeenIDs: %w", err)
	}
	return nil
}

func (s *Store) RetireHarvestSeenIDs(ctx context.Context, source string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if err := s.queries.CloseJobsByPostingID(ctx, sqlc.CloseJobsByPostingIDParams{Source: source, PostingIds: ids}); err != nil {
		return fmt.Errorf("store.RetireHarvestSeenIDs: close jobs: %w", err)
	}
	if err := s.queries.DeleteHarvestSeenIDs(ctx, sqlc.DeleteHarvestSeenIDsParams{Source: source, ExternalIds: ids}); err != nil {
		return fmt.Errorf("store.RetireHarvestSeenIDs: delete ids: %w", err)
	}
	return nil
}

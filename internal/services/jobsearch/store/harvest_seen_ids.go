package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

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

func (s *Store) MarkHarvestIDsInSitemap(ctx context.Context, source string, ids []string, seenAt time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	err := s.queries.MarkHarvestIDsInSitemap(ctx, sqlc.MarkHarvestIDsInSitemapParams{
		Source: source, ExternalIds: ids, SeenAt: pgtype.Timestamptz{Time: seenAt, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("store.MarkHarvestIDsInSitemap: %w", err)
	}
	return nil
}

func (s *Store) CloseJobsAbsentFromSitemap(ctx context.Context, source string, absentSince time.Time) error {
	err := s.queries.CloseJobsAbsentFromSitemap(ctx, sqlc.CloseJobsAbsentFromSitemapParams{
		Source: source, AbsentSince: pgtype.Timestamptz{Time: absentSince, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("store.CloseJobsAbsentFromSitemap: %w", err)
	}
	return nil
}

func (s *Store) HarvestCompanyCounts(ctx context.Context, source string) (map[string]int, error) {
	rows, err := s.queries.ListHarvestCompanyCounts(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("store.HarvestCompanyCounts: %w", err)
	}
	counts := make(map[string]int, len(rows))
	for _, r := range rows {
		counts[r.Company] = int(r.LiveJobs)
	}
	return counts, nil
}

func (s *Store) RecordHarvestCompanyCounts(ctx context.Context, source string, counts map[string]int) error {
	if len(counts) == 0 {
		return nil
	}
	params := sqlc.RecordHarvestCompanyCountsParams{Source: source}
	for company, n := range counts {
		params.Companies = append(params.Companies, company)
		params.LiveJobs = append(params.LiveJobs, int64(n))
	}
	if err := s.queries.RecordHarvestCompanyCounts(ctx, params); err != nil {
		return fmt.Errorf("store.RecordHarvestCompanyCounts: %w", err)
	}
	return nil
}

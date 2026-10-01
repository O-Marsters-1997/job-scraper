package store

import (
	"context"
	"fmt"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func (s *Store) SyncWTTJSitemap(ctx context.Context, jobIDs, companyNames []string) (dto.SitemapDiff, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.SitemapDiff{}, fmt.Errorf("store.SyncWTTJSitemap: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.queries.WithTx(tx)

	live, err := q.CountLiveWTTJJobs(ctx)
	if err != nil {
		return dto.SitemapDiff{}, fmt.Errorf("store.SyncWTTJSitemap: count live: %w", err)
	}
	upserted, err := q.UpsertWTTJJobs(ctx, jobIDs)
	if err != nil {
		return dto.SitemapDiff{}, fmt.Errorf("store.SyncWTTJSitemap: upsert jobs: %w", err)
	}
	var diff dto.SitemapDiff
	for _, row := range upserted {
		if row.Inserted {
			diff.New = append(diff.New, row.JobID)
		}
	}
	if diff.NewCompanies, err = q.InsertWTTJCompanies(ctx, companyNames); err != nil {
		return dto.SitemapDiff{}, fmt.Errorf("store.SyncWTTJSitemap: insert companies: %w", err)
	}
	if int64(len(upserted))*2 < live {
		diff.GoneSkipped = true
	} else if diff.Gone, err = q.MarkWTTJJobsGone(ctx, jobIDs); err != nil {
		return dto.SitemapDiff{}, fmt.Errorf("store.SyncWTTJSitemap: mark gone: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return dto.SitemapDiff{}, fmt.Errorf("store.SyncWTTJSitemap: %w", err)
	}
	return diff, nil
}

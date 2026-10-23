package store

import (
	"context"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store/sqlc"
)

func insertScrapeRun(ctx context.Context, queries *sqlc.Queries, target sqlc.SourceTarget, trigger dto.ScrapeRunTrigger) error {
	err := queries.InsertScrapeRun(ctx, sqlc.InsertScrapeRunParams{
		ID: target.RunID, Source: target.Source, SourceTargetID: target.ID, Trigger: string(trigger),
	})
	if err != nil {
		return fmt.Errorf("insert scrape run: %w", err)
	}
	return nil
}

type cardTally struct{ total, fresh, elsewhere, repeat int32 }

func (t *cardTally) add(kind string) {
	t.total++
	switch kind {
	case "repeat":
		t.repeat++
	case "elsewhere":
		t.elsewhere++
	default:
		t.fresh++
	}
}

func (s *Store) ScrapeRunPageDone(ctx context.Context, runID string) error {
	return s.scrapeRunCall(ctx, runID, "ScrapeRunPageDone", s.queries.ScrapeRunPageDone)
}

func (s *Store) ScrapeRunDeferred(ctx context.Context, runID string) error {
	return s.scrapeRunCall(ctx, runID, "ScrapeRunDeferred", s.queries.ScrapeRunDeferred)
}

func (s *Store) scrapeRunCall(ctx context.Context, runID, op string, call func(context.Context, pgtype.UUID) error) error {
	id, err := data.UUID(runID)
	if err != nil {
		return err
	}
	if err := call(ctx, id); err != nil {
		return fmt.Errorf("store.%s: %w", op, err)
	}
	return nil
}

func (s *Store) ScrapeRunAttemptFailed(ctx context.Context, runID string, page, attempt int, message string) error {
	id, err := data.UUID(runID)
	if err != nil {
		return err
	}
	if page < 0 || page > math.MaxInt32 || attempt < 0 || attempt > math.MaxInt32 {
		return fmt.Errorf("store.ScrapeRunAttemptFailed: page %d or attempt %d out of range", page, attempt)
	}
	err = s.queries.ScrapeRunAttemptFailed(ctx, sqlc.ScrapeRunAttemptFailedParams{
		RunID: id, Page: int32(page), Attempt: int32(attempt), Message: message,
	})
	if err != nil {
		return fmt.Errorf("store.ScrapeRunAttemptFailed: %w", err)
	}
	return nil
}

func (s *Store) FinishScrapeRun(ctx context.Context, runID string, stop dto.StopReason) error {
	id, err := data.UUID(runID)
	if err != nil {
		return err
	}
	err = s.queries.FinishScrapeRun(ctx, sqlc.FinishScrapeRunParams{ID: id, StopReason: pgtype.Text{String: string(stop), Valid: true}})
	if err != nil {
		return fmt.Errorf("store.FinishScrapeRun: %w", err)
	}
	return nil
}

func (s *Store) DeleteExpiredScrapeRuns(ctx context.Context) error {
	if err := s.queries.DeleteExpiredScrapeRuns(ctx); err != nil {
		return fmt.Errorf("store.DeleteExpiredScrapeRuns: %w", err)
	}
	return nil
}

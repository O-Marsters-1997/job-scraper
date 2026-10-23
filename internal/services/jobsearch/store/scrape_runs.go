package store

import (
	"context"
	"encoding/json"
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

func (s *Store) ListScrapeRuns(ctx context.Context, userID string, opts dto.ScrapeRunListOptions) ([]dto.ScrapeRun, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, ErrInvalidID
	}
	if opts.Limit < 1 || opts.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("store.ListScrapeRuns: limit %d out of range", opts.Limit)
	}
	params := sqlc.ListScrapeRunsParams{UserID: uid, RowLimit: int32(opts.Limit)}
	if opts.TargetID != "" {
		if params.TargetID, err = data.UUID(opts.TargetID); err != nil {
			return nil, ErrInvalidID
		}
	}
	if opts.BeforeID != "" {
		if params.BeforeID, err = data.UUID(opts.BeforeID); err != nil {
			return nil, ErrInvalidID
		}
		params.BeforeStartedAt = pgtype.Timestamptz{Time: opts.BeforeStartedAt, Valid: true}
	}
	rows, err := s.queries.ListScrapeRuns(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("store.ListScrapeRuns: %w", err)
	}
	runs := make([]dto.ScrapeRun, len(rows))
	ids := make([]pgtype.UUID, len(rows))
	index := make(map[string]int, len(rows))
	for i, row := range rows {
		runs[i] = toScrapeRunDTO(row)
		ids[i] = row.ID
		index[runs[i].ID] = i
	}
	errRows, err := s.queries.ListScrapeRunErrors(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("store.ListScrapeRuns errors: %w", err)
	}
	for _, e := range errRows {
		i := index[e.RunID.String()]
		runs[i].Errors = append(runs[i].Errors, dto.ScrapeRunError{
			Page: int(e.Page), Attempt: int(e.Attempt), Message: e.Message, At: e.At.Time,
		})
	}
	return runs, nil
}

func toScrapeRunDTO(row sqlc.ListScrapeRunsRow) dto.ScrapeRun {
	run := dto.ScrapeRun{
		ID:            row.ID.String(),
		Kind:          dto.ScrapeRunKind(row.Kind),
		Source:        row.Source,
		TargetID:      row.SourceTargetID.String(),
		TargetValue:   row.TargetValue,
		TargetFilters: map[string]string{},
		Trigger:       dto.ScrapeRunTrigger(row.Trigger),
		StopReason:    dto.StopReason(row.StopReason.String),
		StartedAt:     row.StartedAt.Time,
		FinishedAt:    data.TimePtr(row.FinishedAt),
		Pages:         int(row.Pages),
		Listings:      int(row.Listings),
		New:           int(row.NewListings),
		Elsewhere:     int(row.Elsewhere),
		Repeat:        int(row.RepeatListings),
		Deferred:      int(row.Deferred),
		Retries:       int(row.Retries),
		Errors:        []dto.ScrapeRunError{},
	}
	if len(row.TargetFilters) > 0 {
		_ = json.Unmarshal(row.TargetFilters, &run.TargetFilters)
	}
	switch {
	case run.FinishedAt == nil:
		run.Outcome = dto.OutcomeRunning
	case run.StopReason == dto.StopFailed:
		run.Outcome = dto.OutcomeFailed
	case run.Retries > 0:
		run.Outcome = dto.OutcomeRetried
	default:
		run.Outcome = dto.OutcomeSucceeded
	}
	return run
}

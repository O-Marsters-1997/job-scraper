package store

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

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

func boardRunParams(run dto.BoardRunStart) (id, boardID pgtype.UUID, err error) {
	if id, err = data.UUID(run.ID); err != nil {
		return id, boardID, err
	}
	boardID, err = data.UUID(run.BoardID)
	return id, boardID, err
}

func (s *Store) BoardRunAttemptFailed(ctx context.Context, run dto.BoardRunStart, attempt int, message string) error {
	id, boardID, err := boardRunParams(run)
	if err != nil {
		return err
	}
	if attempt < 0 || attempt > math.MaxInt32 {
		return fmt.Errorf("store.BoardRunAttemptFailed: attempt %d out of range", attempt)
	}
	err = s.queries.BoardRunAttemptFailed(ctx, sqlc.BoardRunAttemptFailedParams{
		ID: id, Source: run.Source, BoardID: boardID, Trigger: string(run.Trigger),
		StartedAt: pgtype.Timestamptz{Time: run.StartedAt, Valid: true},
		Attempt:   int32(attempt), Message: message,
	})
	if err != nil {
		return fmt.Errorf("store.BoardRunAttemptFailed: %w", err)
	}
	return nil
}

func (s *Store) FinishBoardRun(ctx context.Context, run dto.BoardRunStart, stop dto.StopReason, tally dto.BoardTally) error {
	id, boardID, err := boardRunParams(run)
	if err != nil {
		return err
	}
	params := sqlc.FinishBoardRunParams{
		ID: id, Source: run.Source, BoardID: boardID, Trigger: string(run.Trigger),
		StartedAt:  pgtype.Timestamptz{Time: run.StartedAt, Valid: true},
		StopReason: pgtype.Text{String: string(stop), Valid: true},
	}
	counts := map[dto.JobChangeKind]int{}
	for _, c := range tally.Changes {
		counts[c.Kind]++
	}
	for _, c := range []struct {
		n   int
		dst *int32
	}{{tally.Listed, &params.Listings}, {counts[dto.JobChangeNew], &params.NewListings}, {tally.Elsewhere, &params.Elsewhere}, {counts[dto.JobChangeRemoved], &params.Removed}, {counts[dto.JobChangeReopened], &params.Reopened}} {
		if c.n < 0 || c.n > math.MaxInt32 {
			return fmt.Errorf("store.FinishBoardRun: count %d out of range", c.n)
		}
		*c.dst = int32(c.n)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store.FinishBoardRun: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.queries.WithTx(tx)
	inserted, err := q.FinishBoardRun(ctx, params)
	if err != nil {
		return fmt.Errorf("store.FinishBoardRun: %w", err)
	}
	if inserted == 1 && len(tally.Changes) > 0 {
		if err := insertJobChanges(ctx, q, id, tally.Changes); err != nil {
			return fmt.Errorf("store.FinishBoardRun: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store.FinishBoardRun: commit: %w", err)
	}
	return nil
}

func insertJobChanges(ctx context.Context, q *sqlc.Queries, runID pgtype.UUID, changes []dto.JobChange) error {
	type row struct {
		JobID    string            `json:"job_id"`
		Change   dto.JobChangeKind `json:"change"`
		Snapshot json.RawMessage   `json:"snapshot"`
	}
	rows := make([]row, len(changes))
	for i, c := range changes {
		jobID, err := data.UUID(c.JobID)
		if err != nil {
			return err
		}
		got, err := q.GetJob(ctx, sqlc.GetJobParams{ID: jobID})
		if err != nil {
			return data.QueryErr("GetJob", err)
		}
		job, err := toGetJobDTO(got)
		if err != nil {
			return fmt.Errorf("snapshot job: %w", err)
		}
		snapshot, err := json.Marshal(job)
		if err != nil {
			return fmt.Errorf("marshal snapshot: %w", err)
		}
		rows[i] = row{JobID: c.JobID, Change: c.Kind, Snapshot: snapshot}
	}
	payload, err := json.Marshal(rows)
	if err != nil {
		return fmt.Errorf("marshal job changes: %w", err)
	}
	if err := q.InsertScrapeRunJobChanges(ctx, sqlc.InsertScrapeRunJobChangesParams{RunID: runID, Changes: payload}); err != nil {
		return fmt.Errorf("insert job changes: %w", err)
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

func (s *Store) ListScrapeRunJobSnapshots(ctx context.Context, userID, runID string) ([]dto.ScrapeRunJobSnapshot, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, ErrInvalidID
	}
	rid, err := data.UUID(runID)
	if err != nil {
		return nil, ErrInvalidID
	}
	tracked, err := s.queries.ScrapeRunTracked(ctx, sqlc.ScrapeRunTrackedParams{UserID: uid, RunID: rid})
	if err != nil {
		return nil, fmt.Errorf("store.ListScrapeRunJobSnapshots: %w", err)
	}
	if !tracked {
		return nil, data.ErrNotFound
	}
	rows, err := s.queries.ListScrapeRunJobSnapshots(ctx, rid)
	if err != nil {
		return nil, fmt.Errorf("store.ListScrapeRunJobSnapshots: %w", err)
	}
	out := make([]dto.ScrapeRunJobSnapshot, len(rows))
	for i, r := range rows {
		out[i] = dto.ScrapeRunJobSnapshot{JobID: r.JobID.String(), Kind: dto.JobChangeKind(r.Change), Snapshot: r.Snapshot}
	}
	return out, nil
}

func (s *Store) ListScrapeRuns(ctx context.Context, userID string, opts dto.ScrapeRunListOptions) ([]dto.ScrapeRun, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, ErrInvalidID
	}
	if opts.Limit < 1 || opts.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("store.ListScrapeRuns: limit %d out of range", opts.Limit)
	}
	f, err := toRunFilter(opts.ScrapeRunFilter)
	if err != nil {
		return nil, err
	}
	params := sqlc.ListScrapeRunsParams{
		UserID: uid, TargetID: f.target, CompanyID: f.company, FromAt: f.from, ToAt: f.to, Q: f.q,
		Outcomes: opts.Outcome, Kinds: opts.Kind, Sources: opts.Source, Triggers: opts.Trigger,
		RowLimit: int32(opts.Limit),
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
	changeRows, err := s.queries.ListScrapeRunJobChanges(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("store.ListScrapeRuns changes: %w", err)
	}
	for _, c := range changeRows {
		i := index[c.RunID.String()]
		runs[i].Changes = append(runs[i].Changes, dto.ScrapeRunJobChange{
			JobID: c.JobID.String(), Kind: dto.JobChangeKind(c.Change), Title: c.Title,
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
		BoardID:       row.BoardID.String(),
		CompanyID:     row.CompanyID.String(),
		CompanyName:   row.CompanyName,
		Trigger:       dto.ScrapeRunTrigger(row.Trigger),
		StopReason:    dto.StopReason(row.StopReason.String),
		StartedAt:     row.StartedAt.Time,
		FinishedAt:    data.TimePtr(row.FinishedAt),
		Pages:         int(row.Pages),
		Listings:      int(row.Listings),
		New:           int(row.NewListings),
		Elsewhere:     int(row.Elsewhere),
		Repeat:        int(row.RepeatListings),
		Removed:       int(row.Removed),
		Reopened:      int(row.Reopened),
		Deferred:      int(row.Deferred),
		Retries:       int(row.Retries),
		Errors:        []dto.ScrapeRunError{},
		Changes:       []dto.ScrapeRunJobChange{},
	}
	if len(row.TargetFilters) > 0 {
		_ = json.Unmarshal(row.TargetFilters, &run.TargetFilters)
	}
	run.Outcome = scrapeRunOutcome(run.FinishedAt != nil, run.StopReason, run.Retries)
	return run
}

func scrapeRunOutcome(finished bool, stop dto.StopReason, retries int) dto.ScrapeRunOutcome {
	switch {
	case !finished:
		return dto.OutcomeRunning
	case stop == dto.StopFailed:
		return dto.OutcomeFailed
	case retries > 0:
		return dto.OutcomeRetried
	default:
		return dto.OutcomeSucceeded
	}
}

func (s *Store) ListRecentNewJobs(ctx context.Context, userID string, since time.Time) ([]dto.SourceNewJob, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, ErrInvalidID
	}
	rows, err := s.queries.ListRecentNewJobSnapshots(ctx, sqlc.ListRecentNewJobSnapshotsParams{
		Since: pgtype.Timestamptz{Time: since, Valid: true}, UserID: uid,
	})
	if err != nil {
		return nil, fmt.Errorf("store.ListRecentNewJobs: %w", err)
	}
	out := make([]dto.SourceNewJob, len(rows))
	for i, r := range rows {
		out[i] = dto.SourceNewJob{Job: dto.Job{Title: r.Title, Location: r.Location, CompanySlug: r.CompanySlug}}
		if r.SourceTargetID.Valid {
			out[i].TargetID = r.SourceTargetID.String()
		}
		if r.BoardID.Valid {
			out[i].BoardID = r.BoardID.String()
		}
	}
	return out, nil
}

func (s *Store) ListSourceHealth(ctx context.Context, userID string) ([]dto.SourceHealth, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, ErrInvalidID
	}
	rows, err := s.queries.ListSourceHealthRuns(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListSourceHealth: %w", err)
	}
	var out []dto.SourceHealth
	for _, row := range rows {
		id := row.SourceTargetID.String()
		if len(out) == 0 || out[len(out)-1].TargetID != id {
			h := dto.SourceHealth{
				Kind: dto.ScrapeRunSearch, TargetID: id, Source: row.Source, TargetValue: row.TargetValue,
				TargetFilters: map[string]string{}, Recent: []dto.ScrapeRunOutcome{},
			}
			if len(row.TargetFilters) > 0 {
				_ = json.Unmarshal(row.TargetFilters, &h.TargetFilters)
			}
			out = append(out, h)
		}
		if !row.StartedAt.Valid {
			continue
		}
		h := &out[len(out)-1]
		h.Recent = append(h.Recent, scrapeRunOutcome(true, dto.StopReason(row.StopReason.String), int(row.Retries)))
		h.LastRunAt = &row.StartedAt.Time
		h.New += int(row.NewListings)
		h.Elsewhere += int(row.Elsewhere)
		h.Retries += int(row.Retries)
	}
	boardRows, err := s.queries.ListBoardHealthRuns(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListSourceHealth boards: %w", err)
	}
	searches := len(out)
	for _, row := range boardRows {
		id := row.BoardID.String()
		if len(out) == searches || out[len(out)-1].BoardID != id {
			out = append(out, dto.SourceHealth{
				Kind: dto.ScrapeRunBoard, BoardID: id, Source: row.Source, CompanyName: row.CompanyName,
				TargetFilters: map[string]string{}, Recent: []dto.ScrapeRunOutcome{},
			})
		}
		if !row.StartedAt.Valid {
			continue
		}
		h := &out[len(out)-1]
		h.Recent = append(h.Recent, scrapeRunOutcome(true, dto.StopReason(row.StopReason.String), int(row.Retries)))
		h.LastRunAt = &row.StartedAt.Time
		h.New += int(row.NewListings)
		h.Elsewhere += int(row.Elsewhere)
		h.Removed += int(row.Removed)
		h.Retries += int(row.Retries)
	}
	return out, nil
}

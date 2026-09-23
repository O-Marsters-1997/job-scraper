package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func (db *DB) ListDueBoards(ctx context.Context) ([]dto.BoardPoll, error) {
	rows, err := db.queries.ListDueBoards(ctx)
	if err != nil {
		return nil, fmt.Errorf("list due boards: %w", err)
	}
	boards := make([]dto.BoardPoll, len(rows))
	for i, row := range rows {
		boards[i] = dto.BoardPoll{ID: row.ID.String(), CompanyID: row.CompanyID.String(), CompanySlug: row.CompanySlug, Source: row.Source, Token: row.BoardToken, IntervalMinutes: int(row.IntervalMinutes)}
	}
	return boards, nil
}

func (db *DB) ListActiveBoards(ctx context.Context) ([]dto.BoardPoll, error) {
	rows, err := db.queries.ListActiveBoards(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active boards: %w", err)
	}
	boards := make([]dto.BoardPoll, len(rows))
	for i, row := range rows {
		boards[i] = dto.BoardPoll{ID: row.ID.String(), CompanyID: row.CompanyID.String(), CompanySlug: row.CompanySlug, Source: row.Source, Token: row.BoardToken, IntervalMinutes: int(row.IntervalMinutes)}
	}
	return boards, nil
}

func (db *DB) GetVerifiedBoardID(ctx context.Context, source, token string) (string, error) {
	id, err := db.queries.GetVerifiedBoardID(ctx, pgsqlc.GetVerifiedBoardIDParams{Source: source, BoardToken: token})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", providers.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get verified board: %w", err)
	}
	return id.String(), nil
}

func (db *DB) ClaimBoard(ctx context.Context, id string, manual bool) (dto.BoardPoll, error) {
	boardID, err := parseUUID(id)
	if err != nil {
		return dto.BoardPoll{}, err
	}
	board, err := db.queries.GetPollBoard(ctx, boardID)
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.BoardPoll{}, providers.ErrBoardClaimUnavailable
	}
	if err != nil {
		return dto.BoardPoll{}, fmt.Errorf("get poll board: %w", err)
	}
	if !manual && board.LastScheduledAt.Valid && time.Since(board.LastScheduledAt.Time) < time.Duration(board.IntervalMinutes)*time.Minute {
		return dto.BoardPoll{}, providers.ErrBoardClaimUnavailable
	}
	if err := db.queries.EnsureBoardPollState(ctx, boardID); err != nil {
		return dto.BoardPoll{}, fmt.Errorf("ensure poll state: %w", err)
	}
	owner := uuid.NewString()
	claim, err := db.queries.ClaimPollState(ctx, pgsqlc.ClaimPollStateParams{BoardID: boardID, LeaseOwner: pgtype.Text{String: owner, Valid: true}, Manual: manual})
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.BoardPoll{}, providers.ErrBoardClaimUnavailable
	}
	if err != nil {
		return dto.BoardPoll{}, fmt.Errorf("claim poll state: %w", err)
	}
	return dto.BoardPoll{ID: id, CompanyID: board.CompanyID.String(), CompanySlug: board.CompanySlug, Source: board.Source, Token: board.BoardToken, IntervalMinutes: int(board.IntervalMinutes), LeaseOwner: owner, Version: claim.LastSnapshotVersion, StartedAt: claim.LastStartedAt.Time, Manual: manual}, nil
}

func (db *DB) FailBoard(ctx context.Context, poll dto.BoardPoll) error {
	id, err := parseUUID(poll.ID)
	if err != nil {
		return err
	}
	n, err := db.queries.FailPollState(ctx, pgsqlc.FailPollStateParams{BoardID: id, LeaseOwner: pgtype.Text{String: poll.LeaseOwner, Valid: true}, LastSnapshotVersion: poll.Version})
	if err != nil {
		return fmt.Errorf("fail board: %w", err)
	}
	if n != 1 {
		return providers.ErrBoardClaimUnavailable
	}
	return nil
}

func (db *DB) CompleteBoard(ctx context.Context, snapshot dto.BoardSnapshot) error {
	if !snapshot.Complete {
		return errors.New("incomplete board snapshot")
	}
	poll := snapshot.Poll
	id, err := parseUUID(poll.ID)
	if err != nil {
		return err
	}
	urls := make([]string, 0, len(snapshot.Jobs))
	seenURLs := make(map[string]bool, len(snapshot.Jobs))
	for _, job := range snapshot.Jobs {
		url, err := normalizeJobURL(job.URL)
		if err != nil {
			return err
		}
		if !seenURLs[url] {
			urls = append(urls, url)
			seenURLs[url] = true
		}
	}
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin board completion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := db.queries.WithTx(tx)
	state, err := q.LockPollState(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return providers.ErrBoardClaimUnavailable
	}
	if err != nil {
		return fmt.Errorf("lock board state: %w", err)
	}
	if state.LastSnapshotVersion != poll.Version || !state.LeaseOwner.Valid || state.LeaseOwner.String != poll.LeaseOwner || !state.LeaseUntil.Valid || !state.LeaseUntil.Time.After(time.Now()) {
		return providers.ErrBoardClaimUnavailable
	}
	aliases, err := q.FindBoardJobAliases(ctx, urls)
	if err != nil {
		return fmt.Errorf("resolve board jobs: %w", err)
	}
	if len(aliases) != len(urls) {
		return errors.New("board ingest incomplete: job URL missing")
	}
	seenJobs := make(map[pgtype.UUID]bool, len(aliases))
	for _, alias := range aliases {
		if seenJobs[alias.JobID] {
			continue
		}
		seenJobs[alias.JobID] = true
		if err := q.ObserveBoardJob(ctx, pgsqlc.ObserveBoardJobParams{BoardID: id, JobID: alias.JobID, LastSnapshotVersion: poll.Version}); err != nil {
			return fmt.Errorf("observe board job: %w", err)
		}
	}
	if err := q.ReopenObservedBoardJobs(ctx, pgsqlc.ReopenObservedBoardJobsParams{BoardID: id, LastSnapshotVersion: poll.Version}); err != nil {
		return fmt.Errorf("reopen board jobs: %w", err)
	}
	if len(urls) > 0 || state.ConsecutiveCompleteEmpty >= 1 {
		if err := q.CloseMissingBoardJobs(ctx, pgsqlc.CloseMissingBoardJobsParams{PrimaryBoardID: id, LastSnapshotVersion: poll.Version}); err != nil {
			return fmt.Errorf("close missing board jobs: %w", err)
		}
	}
	if len(urls) == 0 && state.ConsecutiveCompleteEmpty >= 1 {
		if err := q.RetireSupersededBoard(ctx, id); err != nil {
			return fmt.Errorf("retire board: %w", err)
		}
	}
	n, err := q.CompletePollState(ctx, pgsqlc.CompletePollStateParams{Manual: poll.Manual, IntervalMinutes: int32(poll.IntervalMinutes), Empty: len(urls) == 0, BoardID: id, LeaseOwner: poll.LeaseOwner, Version: poll.Version})
	if err != nil {
		return fmt.Errorf("complete board state: %w", err)
	}
	if n != 1 {
		return providers.ErrBoardClaimUnavailable
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit board completion: %w", err)
	}
	return nil
}

package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

var _ providers.ScoringEffectsProvider = (*DB)(nil)

func (db *DB) ClaimScoringEffect(ctx context.Context) (dto.ScoringEffect, error) {
	row, err := db.queries.ClaimScoringEffect(ctx)
	if err != nil {
		return dto.ScoringEffect{}, err
	}
	return dto.ScoringEffect{
		ID: row.ID.String(), JobID: row.JobID.String(), UserID: row.UserID.String(),
		Fingerprint: row.Fingerprint, ConfigVersion: row.ConfigVersion.Time,
		Model: row.Model, Attempts: int(row.Attempts), FirstDiscovery: row.FirstDiscovery,
	}, nil
}

func (db *DB) FailScoringEffect(ctx context.Context, id string, attempts int, failure dto.ScoringFailure) error {
	effectID, err := parseUUID(id)
	if err != nil {
		return err
	}
	var retryAfterSecs pgtype.Int4
	if failure.RetryAfter > 0 {
		retryAfterSecs = pgtype.Int4{Int32: int32(failure.RetryAfter.Seconds()), Valid: true}
	}
	params := pgsqlc.FailScoringEffectParams{
		ID: effectID, Attempts: int32(attempts), LastError: failure.Reason,
		Terminal: failure.Terminal, RetryAfterSecs: retryAfterSecs,
	}
	if err := db.queries.FailScoringEffect(ctx, params); err != nil {
		return fmt.Errorf("fail scoring effect: %w", err)
	}
	return nil
}

func (db *DB) CompleteScoringEffect(ctx context.Context, effect dto.ScoringEffect, score int, reasoning string, matched, missing []string) (bool, error) {
	effectID, err := parseUUID(effect.ID)
	if err != nil {
		return false, err
	}
	rows, err := db.queries.CompleteScoringEffect(ctx, pgsqlc.CompleteScoringEffectParams{
		ID: effectID, Attempts: int32(effect.Attempts), Score: int32(score), Reasoning: reasoning,
		Matched: matched, Missing: missing,
	})
	if err != nil {
		return false, fmt.Errorf("complete scoring effect: %w", err)
	}
	return rows == 1, nil
}

// ScoringStatus aliases dto.ScoringStatus.
type ScoringStatus = dto.ScoringStatus

func (db *DB) GetScoringStatus(ctx context.Context, userID string) (ScoringStatus, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return ScoringStatus{}, err
	}
	row, err := db.queries.GetScoringStatus(ctx, uid)
	if err != nil {
		return ScoringStatus{}, fmt.Errorf("get scoring status: %w", err)
	}
	return ScoringStatus{Pending: row.Pending, Failed: row.Failed, Stale: row.Stale}, nil
}

func (db *DB) QueueRescore(ctx context.Context, userID string) (int64, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return 0, err
	}
	count, err := db.queries.QueueRescore(ctx, uid)
	if err != nil {
		return 0, fmt.Errorf("queue rescore: %w", err)
	}
	return count, nil
}

func (db *DB) QueueTrackingScores(ctx context.Context, userID, companyID string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	cid, err := parseUUID(companyID)
	if err != nil {
		return err
	}
	if err := db.queries.QueueTrackingScores(ctx, pgsqlc.QueueTrackingScoresParams{UserID: uid, CompanyID: cid}); err != nil {
		return fmt.Errorf("queue tracking scores: %w", err)
	}
	return nil
}

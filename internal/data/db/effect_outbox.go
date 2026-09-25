package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/score"
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

func (db *DB) CompleteScoringEffect(ctx context.Context, effect dto.ScoringEffect, result score.SuitabilityResult) (bool, error) {
	effectID, err := parseUUID(effect.ID)
	if err != nil {
		return false, err
	}
	criteria, err := json.Marshal(result.Criteria)
	if err != nil {
		return false, fmt.Errorf("complete scoring effect: marshal criteria: %w", err)
	}
	cost, err := toNumeric(result.Cost)
	if err != nil {
		return false, fmt.Errorf("complete scoring effect: cost: %w", err)
	}
	rows, err := db.queries.CompleteScoringEffect(ctx, pgsqlc.CompleteScoringEffectParams{
		ID: effectID, Attempts: int32(effect.Attempts), Score: int32(result.Score),
		Criteria:     criteria,
		Confidence:   pgtype.Float4{Float32: float32(result.Confidence), Valid: true},
		Cost:         cost,
		ScoreModel:   result.Model,
		CurrentModel: score.JevModel,
	})
	if err != nil {
		return false, fmt.Errorf("complete scoring effect: %w", err)
	}
	return rows == 1, nil
}

type ScoringStatus = dto.ScoringStatus

func (db *DB) GetScoringStatus(ctx context.Context, userID string) (ScoringStatus, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return ScoringStatus{}, err
	}
	row, err := db.queries.GetScoringStatus(ctx, pgsqlc.GetScoringStatusParams{UserID: uid, Model: score.JevModel})
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
	count, err := db.queries.QueueRescore(ctx, pgsqlc.QueueRescoreParams{UserID: uid, Model: score.JevModel})
	if err != nil {
		return 0, fmt.Errorf("queue rescore: %w", err)
	}
	return count, nil
}

type scoringEffectsInput struct {
	Job            dto.Job
	JobID          pgtype.UUID
	CompanyID      pgtype.UUID
	Discovery      bool
	FirstDiscovery bool
}

func queueScoringEffects(ctx context.Context, queries *pgsqlc.Queries, in scoringEffectsInput) error {
	job := in.Job
	users, err := queries.FindInterestedUsers(ctx, pgsqlc.FindInterestedUsersParams{
		CompanyID: in.CompanyID, CompanySlug: job.CompanySlug, Source: job.Source, Discovery: in.Discovery,
	})
	if err != nil {
		return fmt.Errorf("find interested users: %w", err)
	}
	for _, user := range users {
		cfg := dto.SearchConfig{
			ExcludedTitleKeywords: user.ExcludedTitleKeywords,
			ExcludedCompanies:     user.ExcludedCompanies,
			ExcludedSeniority:     user.ExcludedSeniority,
			ExcludedLocations:     user.ExcludedLocations,
		}
		if _, rejected := score.Reject(job, cfg); rejected {
			continue
		}
		hasCredential, err := queries.HasUserAICredential(ctx, pgsqlc.HasUserAICredentialParams{
			UserID: user.UserID, Provider: score.Provider,
		})
		if err != nil {
			return fmt.Errorf("check scoring credential: %w", err)
		}
		if !hasCredential {
			continue
		}
		if err := queries.InsertScoringEffect(ctx, pgsqlc.InsertScoringEffectParams{
			JobID: in.JobID, UserID: user.UserID, Fingerprint: job.ContentFingerprint,
			ConfigVersion: user.ConfigVersion, Model: score.JevModel, FirstDiscovery: in.FirstDiscovery,
		}); err != nil {
			return fmt.Errorf("insert scoring effect: %w", err)
		}
	}
	return nil
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
	params := pgsqlc.QueueTrackingScoresParams{UserID: uid, CompanyID: cid, Model: score.JevModel}
	if err := db.queries.QueueTrackingScores(ctx, params); err != nil {
		return fmt.Errorf("queue tracking scores: %w", err)
	}
	return nil
}

func toNumeric(f float64) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if err := n.ScanScientific(strconv.FormatFloat(f, 'f', -1, 64)); err != nil {
		return pgtype.Numeric{}, err
	}
	return n, nil
}

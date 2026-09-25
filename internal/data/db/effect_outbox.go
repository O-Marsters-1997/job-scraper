package db

import (
	"context"
	"fmt"

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

func (db *DB) FailScoringEffect(ctx context.Context, id string, attempts int, reason string) error {
	effectID, err := parseUUID(id)
	if err != nil {
		return err
	}
	if err := db.queries.FailScoringEffect(ctx, pgsqlc.FailScoringEffectParams{ID: effectID, Attempts: int32(attempts), LastError: reason}); err != nil {
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

// ScoringStatus is an alias so existing callers keep compiling; the type
// itself lives in dto so internal/data/providers can declare an interface
// against it without importing db.
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

func queueScoringEffects(ctx context.Context, queries *pgsqlc.Queries, job dto.Job, jobID, companyID pgtype.UUID, discovery, firstDiscovery bool) error {
	users, err := queries.FindInterestedUsers(ctx, pgsqlc.FindInterestedUsersParams{
		CompanyID: companyID, CompanySlug: job.CompanySlug, Source: job.Source, Discovery: discovery,
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
			JobID: jobID, UserID: user.UserID, Fingerprint: job.ContentFingerprint,
			ConfigVersion: user.ConfigVersion, Model: user.Model, FirstDiscovery: firstDiscovery,
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
	if err := db.queries.QueueTrackingScores(ctx, pgsqlc.QueueTrackingScoresParams{UserID: uid, CompanyID: cid}); err != nil {
		return fmt.Errorf("queue tracking scores: %w", err)
	}
	return nil
}

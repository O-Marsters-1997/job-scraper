package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func fromJobScore(row pgsqlc.JobScore) dto.JobScore {
	js := dto.JobScore{
		JobID:              row.JobID.String(),
		UserID:             row.UserID.String(),
		Matched:            row.Matched,
		Missing:            row.Missing,
		SuitabilitySkipped: row.SuitabilitySkipped,
	}
	if row.RelevanceScore.Valid {
		v := int(row.RelevanceScore.Int32)
		js.RelevanceScore = &v
	}
	if row.SuitabilityScore.Valid {
		v := int(row.SuitabilityScore.Int32)
		js.SuitabilityScore = &v
	}
	if row.Reasoning.Valid {
		js.Reasoning = &row.Reasoning.String
	}
	return js
}

func (db *DB) UpsertJobScoreRelevance(ctx context.Context, jobID, userID string, score int) error {
	jid, err := parseUUID(jobID)
	if err != nil {
		return err
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	if err := db.queries.UpsertJobScoreRelevance(ctx, pgsqlc.UpsertJobScoreRelevanceParams{
		JobID:          jid,
		UserID:         uid,
		RelevanceScore: pgtype.Int4{Int32: int32(score), Valid: true},
	}); err != nil {
		return fmt.Errorf("db.UpsertJobScoreRelevance: %w", err)
	}
	return nil
}

func (db *DB) UpsertJobScoreSuitability(ctx context.Context, jobID, userID string, score int, reasoning string, matched, missing []string) error {
	jid, err := parseUUID(jobID)
	if err != nil {
		return err
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	if err := db.queries.UpsertJobScoreSuitability(ctx, pgsqlc.UpsertJobScoreSuitabilityParams{
		JobID:            jid,
		UserID:           uid,
		SuitabilityScore: pgtype.Int4{Int32: int32(score), Valid: true},
		Reasoning:        pgtype.Text{String: reasoning, Valid: reasoning != ""},
		Matched:          matched,
		Missing:          missing,
	}); err != nil {
		return fmt.Errorf("db.UpsertJobScoreSuitability: %w", err)
	}
	return nil
}

func (db *DB) UpsertJobScoreSkipped(ctx context.Context, jobID, userID string) error {
	jid, err := parseUUID(jobID)
	if err != nil {
		return err
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	if err := db.queries.UpsertJobScoreSkipped(ctx, pgsqlc.UpsertJobScoreSkippedParams{
		JobID:  jid,
		UserID: uid,
	}); err != nil {
		return fmt.Errorf("db.UpsertJobScoreSkipped: %w", err)
	}
	return nil
}

func (db *DB) GetJobScore(ctx context.Context, jobID, userID string) (dto.JobScore, error) {
	jid, err := parseUUID(jobID)
	if err != nil {
		return dto.JobScore{}, err
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.JobScore{}, err
	}
	row, err := db.queries.GetJobScore(ctx, pgsqlc.GetJobScoreParams{
		JobID:  jid,
		UserID: uid,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.JobScore{}, providers.ErrNotFound
	}
	if err != nil {
		return dto.JobScore{}, fmt.Errorf("db.GetJobScore: %w", err)
	}
	return fromJobScore(row), nil
}

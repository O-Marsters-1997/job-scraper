package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func fromUserAIPrefs(row pgsqlc.UserAiPref) dto.UserAIPrefs {
	return dto.UserAIPrefs{
		SuitabilityModel: row.SuitabilityModel,
		ReasoningModel:   row.ReasoningModel,
	}
}

func (db *DB) GetUserAIPrefs(ctx context.Context, userID string) (dto.UserAIPrefs, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.UserAIPrefs{}, err
	}
	row, err := db.queries.GetUserAIPrefs(ctx, uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.UserAIPrefs{}, providers.ErrNotFound
	}
	if err != nil {
		return dto.UserAIPrefs{}, fmt.Errorf("db.GetUserAIPrefs: %w", err)
	}
	return fromUserAIPrefs(row), nil
}

func (db *DB) UpsertUserAIPrefs(ctx context.Context, userID, suitabilityModel, reasoningModel string) (dto.UserAIPrefs, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.UserAIPrefs{}, err
	}
	row, err := db.queries.UpsertUserAIPrefs(ctx, pgsqlc.UpsertUserAIPrefsParams{
		UserID:           uid,
		SuitabilityModel: suitabilityModel,
		ReasoningModel:   reasoningModel,
	})
	if err != nil {
		return dto.UserAIPrefs{}, fmt.Errorf("db.UpsertUserAIPrefs: %w", err)
	}
	return fromUserAIPrefs(row), nil
}

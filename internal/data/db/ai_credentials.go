package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
)

func (db *DB) UpsertUserAICredential(ctx context.Context, userID, provider, encKey string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	_, err = db.queries.UpsertUserAICredential(ctx, pgsqlc.UpsertUserAICredentialParams{
		UserID:    uid,
		Provider:  provider,
		ApiKeyEnc: encKey,
	})
	if err != nil {
		return fmt.Errorf("db.UpsertUserAICredential: %w", err)
	}
	return nil
}

func (db *DB) GetUserAICredential(ctx context.Context, userID, provider string) (string, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return "", err
	}
	row, err := db.queries.GetUserAICredential(ctx, pgsqlc.GetUserAICredentialParams{
		UserID:   uid,
		Provider: provider,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", providers.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("db.GetUserAICredential: %w", err)
	}
	return row.ApiKeyEnc, nil
}

func (db *DB) DeleteUserAICredential(ctx context.Context, userID, provider string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	if err := db.queries.DeleteUserAICredential(ctx, pgsqlc.DeleteUserAICredentialParams{
		UserID:   uid,
		Provider: provider,
	}); err != nil {
		return fmt.Errorf("db.DeleteUserAICredential: %w", err)
	}
	return nil
}

func (db *DB) ListUsersWithProvider(ctx context.Context, provider string) ([]string, error) {
	uuids, err := db.queries.ListUsersWithProvider(ctx, provider)
	if err != nil {
		return nil, fmt.Errorf("db.ListUsersWithProvider: %w", err)
	}
	ids := make([]string, len(uuids))
	for i, u := range uuids {
		ids[i] = u.String()
	}
	return ids, nil
}

func (db *DB) ListUserAICredentialProviders(ctx context.Context, userID string) ([]string, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := db.queries.ListUserAICredentialProviders(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("db.ListUserAICredentialProviders: %w", err)
	}
	return rows, nil
}

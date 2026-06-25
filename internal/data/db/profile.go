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

func fromProfile(id pgtype.UUID, username string, email pgtype.Text) dto.Profile {
	return dto.Profile{
		Username: username,
		Email:    email.String,
	}
}

func (db *DB) GetProfile(ctx context.Context, userID string) (dto.Profile, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.Profile{}, err
	}
	row, err := db.queries.GetUserProfile(ctx, uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.Profile{}, providers.ErrNotFound
	}
	if err != nil {
		return dto.Profile{}, fmt.Errorf("db.GetProfile: %w", err)
	}
	return fromProfile(row.ID, row.Username, row.Email), nil
}

func (db *DB) GetUserEmail(ctx context.Context, userID string) (string, error) {
	p, err := db.GetProfile(ctx, userID)
	if err != nil {
		return "", err
	}
	return p.Email, nil
}

func (db *DB) UpdateEmail(ctx context.Context, userID, email string) (dto.Profile, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.Profile{}, err
	}
	row, err := db.queries.UpdateUserEmail(ctx, pgsqlc.UpdateUserEmailParams{
		ID:    uid,
		Email: pgtype.Text{String: email, Valid: email != ""},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.Profile{}, providers.ErrNotFound
	}
	if err != nil {
		return dto.Profile{}, fmt.Errorf("db.UpdateEmail: %w", err)
	}
	return fromProfile(row.ID, row.Username, row.Email), nil
}

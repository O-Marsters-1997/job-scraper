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

func fromSourceTarget(row pgsqlc.SourceTarget) dto.SourceTarget {
	return dto.SourceTarget{
		ID:      row.ID.String(),
		UserID:  row.UserID.String(),
		Source:  row.Source,
		Value:   row.Value,
		Enabled: row.Enabled,
	}
}

func (db *DB) ListSourceTargetsByUser(ctx context.Context, userID string) ([]dto.SourceTarget, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := db.queries.ListSourceTargetsByUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("db.ListSourceTargetsByUser: %w", err)
	}
	out := make([]dto.SourceTarget, len(rows))
	for i, r := range rows {
		out[i] = fromSourceTarget(r)
	}
	return out, nil
}

func (db *DB) ListEnabledSourceTargets(ctx context.Context) ([]dto.SourceTarget, error) {
	rows, err := db.queries.ListEnabledSourceTargets(ctx)
	if err != nil {
		return nil, fmt.Errorf("db.ListEnabledSourceTargets: %w", err)
	}
	out := make([]dto.SourceTarget, len(rows))
	for i, r := range rows {
		out[i] = fromSourceTarget(r)
	}
	return out, nil
}

func (db *DB) CreateSourceTarget(ctx context.Context, userID, source, value string, enabled bool) (dto.SourceTarget, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	row, err := db.queries.CreateSourceTarget(ctx, pgsqlc.CreateSourceTargetParams{
		UserID:  uid,
		Source:  source,
		Value:   value,
		Enabled: enabled,
	})
	if err != nil {
		return dto.SourceTarget{}, fmt.Errorf("db.CreateSourceTarget: %w", err)
	}
	return fromSourceTarget(row), nil
}

func (db *DB) UpdateSourceTarget(ctx context.Context, id, userID string, enabled bool) (dto.SourceTarget, error) {
	tid, err := parseUUID(id)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	row, err := db.queries.UpdateSourceTarget(ctx, pgsqlc.UpdateSourceTargetParams{
		ID:      tid,
		UserID:  uid,
		Enabled: enabled,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dto.SourceTarget{}, providers.ErrNotFound
		}
		return dto.SourceTarget{}, fmt.Errorf("db.UpdateSourceTarget: %w", err)
	}
	return fromSourceTarget(row), nil
}

func (db *DB) DeleteSourceTarget(ctx context.Context, id, userID string) error {
	tid, err := parseUUID(id)
	if err != nil {
		return err
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	if err := db.queries.DeleteSourceTarget(ctx, pgsqlc.DeleteSourceTargetParams{
		ID:     tid,
		UserID: uid,
	}); err != nil {
		return fmt.Errorf("db.DeleteSourceTarget: %w", err)
	}
	return nil
}

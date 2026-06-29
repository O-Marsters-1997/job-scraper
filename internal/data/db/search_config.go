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

func fromSearchConfig(row pgsqlc.SearchConfig) dto.SearchConfig {
	return dto.SearchConfig{
		ID:                row.ID.String(),
		UserID:            row.UserID.String(),
		Role:              row.Role,
		Location:          row.Location,
		Keywords:          row.Keywords,
		SuitabilityRubric: row.SuitabilityRubric,
		RelevanceCutoff:   int(row.RelevanceCutoff),
		NotifyThreshold:   int(row.NotifyThreshold),
	}
}

func (db *DB) ListSearchConfigs(ctx context.Context) ([]dto.SearchConfig, error) {
	rows, err := db.queries.ListSearchConfigs(ctx)
	if err != nil {
		return nil, fmt.Errorf("db.ListSearchConfigs: %w", err)
	}
	cfgs := make([]dto.SearchConfig, len(rows))
	for i, row := range rows {
		cfgs[i] = fromSearchConfig(row)
	}
	return cfgs, nil
}

func (db *DB) GetSearchConfig(ctx context.Context, userID string) (dto.SearchConfig, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.SearchConfig{}, err
	}
	row, err := db.queries.GetSearchConfig(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dto.SearchConfig{}, providers.ErrNotFound
		}
		return dto.SearchConfig{}, fmt.Errorf("db.GetSearchConfig: %w", err)
	}
	return fromSearchConfig(row), nil
}

func (db *DB) UpsertSearchConfig(ctx context.Context, cfg dto.SearchConfig) (dto.SearchConfig, error) {
	uid, err := parseUUID(cfg.UserID)
	if err != nil {
		return dto.SearchConfig{}, err
	}
	keywords := cfg.Keywords
	if keywords == nil {
		keywords = []string{}
	}
	row, err := db.queries.UpsertSearchConfig(ctx, pgsqlc.UpsertSearchConfigParams{
		UserID:            uid,
		Role:              cfg.Role,
		Location:          cfg.Location,
		Keywords:          keywords,
		SuitabilityRubric: cfg.SuitabilityRubric,
		RelevanceCutoff:   int32(cfg.RelevanceCutoff),
		NotifyThreshold:   int32(cfg.NotifyThreshold),
	})
	if err != nil {
		return dto.SearchConfig{}, fmt.Errorf("db.UpsertSearchConfig: %w", err)
	}
	return fromSearchConfig(row), nil
}

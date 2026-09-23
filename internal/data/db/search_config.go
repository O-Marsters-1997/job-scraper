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
		ID:                    row.ID.String(),
		UserID:                row.UserID.String(),
		ExcludedTitleKeywords: row.ExcludedTitleKeywords,
		ExcludedCompanies:     row.ExcludedCompanies,
		ExcludedSeniority:     row.ExcludedSeniority,
		ExcludedLocations:     row.ExcludedLocations,
		SuitabilityRubric:     row.SuitabilityRubric,
		NotifyThreshold:       int(row.NotifyThreshold),
		UpdatedAt:             row.UpdatedAt.Time,
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
	row, err := db.queries.UpsertSearchConfig(ctx, pgsqlc.UpsertSearchConfigParams{
		UserID:                uid,
		ExcludedTitleKeywords: nonNilStrings(cfg.ExcludedTitleKeywords),
		ExcludedCompanies:     nonNilStrings(cfg.ExcludedCompanies),
		ExcludedSeniority:     nonNilStrings(cfg.ExcludedSeniority),
		ExcludedLocations:     nonNilStrings(cfg.ExcludedLocations),
		SuitabilityRubric:     cfg.SuitabilityRubric,
		NotifyThreshold:       int32(cfg.NotifyThreshold),
	})
	if err != nil {
		return dto.SearchConfig{}, fmt.Errorf("db.UpsertSearchConfig: %w", err)
	}
	return fromSearchConfig(row), nil
}

// nonNilStrings replaces a nil slice with empty so pgx encodes it as an
// empty array instead of NULL for a NOT NULL column.
func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

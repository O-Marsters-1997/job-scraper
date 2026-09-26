package db

import (
	"context"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func fromScoringOption(row pgsqlc.ScoringOption) dto.ScoringOption {
	var retiredAt *time.Time
	if row.RetiredAt.Valid {
		t := row.RetiredAt.Time
		retiredAt = &t
	}
	return dto.ScoringOption{
		ID:        row.ID,
		Dimension: dto.Dimension(row.Dimension),
		Label:     row.Label,
		Question:  row.Question,
		RetiredAt: retiredAt,
	}
}

func (db *DB) ListScoringOptions(ctx context.Context) ([]dto.ScoringOption, error) {
	rows, err := db.queries.ListScoringOptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("db.ListScoringOptions: %w", err)
	}
	options := make([]dto.ScoringOption, len(rows))
	for i, row := range rows {
		options[i] = fromScoringOption(row)
	}
	return options, nil
}

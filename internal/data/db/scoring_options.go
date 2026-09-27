package db

import (
	"context"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data"
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

// AddScoringOption inserts a bank option and queues an answer effect for
// every non-closed job that already has a job_scores row, so the new
// question reaches jobs already scored.
func (db *DB) AddScoringOption(ctx context.Context, id, dimension, label, question string) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin add scoring option: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.queries.WithTx(tx)

	err = queries.InsertScoringOption(ctx, pgsqlc.InsertScoringOptionParams{
		ID: id, Dimension: pgsqlc.ScoringDimension(dimension), Label: label, Question: question,
	})
	if err != nil {
		return fmt.Errorf("db.AddScoringOption: %w", err)
	}
	if err := queries.QueueOptionBackfill(ctx); err != nil {
		return fmt.Errorf("db.AddScoringOption: queue backfill: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit add scoring option: %w", err)
	}
	return nil
}

// RewordScoringOption changes an option's question text, which changes its
// question hash, and queues the same backfill as AddScoringOption.
func (db *DB) RewordScoringOption(ctx context.Context, id, question string) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin reword scoring option: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := db.queries.WithTx(tx)

	rows, err := queries.RewordScoringOption(ctx, pgsqlc.RewordScoringOptionParams{ID: id, Question: question})
	if err != nil {
		return fmt.Errorf("db.RewordScoringOption: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("db.RewordScoringOption: option %q: %w", id, data.ErrNotFound)
	}
	if err := queries.QueueOptionBackfill(ctx); err != nil {
		return fmt.Errorf("db.RewordScoringOption: queue backfill: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit reword scoring option: %w", err)
	}
	return nil
}

// RetireScoringOption sets retired_at so the option is hidden from new
// pickers. Existing picks keep scoring from their cached answers.
func (db *DB) RetireScoringOption(ctx context.Context, id string) error {
	rows, err := db.queries.RetireScoringOption(ctx, id)
	if err != nil {
		return fmt.Errorf("db.RetireScoringOption: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("db.RetireScoringOption: option %q: %w", id, data.ErrNotFound)
	}
	return nil
}

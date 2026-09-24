package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

func (db *DB) GetLastScraped(ctx context.Context, source string) (time.Time, bool, error) {
	last, err := db.queries.GetHarvestRun(ctx, source)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, false, nil
	}
	return last.Time, err == nil, err
}

func (db *DB) SetLastScraped(ctx context.Context, source string) error {
	return db.queries.SetHarvestRun(ctx, source)
}

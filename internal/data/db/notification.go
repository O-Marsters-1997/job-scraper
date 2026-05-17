package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
)

func (db *DB) RecordDigest(ctx context.Context, sentAt time.Time, jobCount int) error {
	_, err := db.queries.InsertNotificationDigest(ctx, pgsqlc.InsertNotificationDigestParams{
		SentAt:   pgtype.Timestamptz{Time: sentAt, Valid: true},
		JobCount: int32(jobCount),
	})
	if err != nil {
		return fmt.Errorf("db.RecordDigest: %w", err)
	}
	return nil
}

func (db *DB) GetLastDigestSentAt(ctx context.Context) (time.Time, error) {
	ts, err := db.queries.GetLastNotificationDigestSentAt(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("db.GetLastDigestSentAt: %w", err)
	}
	return ts.Time, nil
}

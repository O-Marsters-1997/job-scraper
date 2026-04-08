package db

import (
	"context"
	"errors"
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
	return err
}

func (db *DB) GetLastDigestSentAt(ctx context.Context) (time.Time, error) {
	ts, err := db.queries.GetLastNotificationDigestSentAt(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	return ts.Time, nil
}

// Package store is the notify context's Postgres store: the notifications table.
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/services/notify/store/sqlc"
)

type Store struct {
	queries *sqlc.Queries
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{queries: sqlc.New(pool)}
}

// InsertNotification records an unread Notification and returns its id.
func (s *Store) InsertNotification(ctx context.Context, userID, jobID, title, body string) (string, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return "", err
	}
	jid, err := data.UUID(jobID)
	if err != nil {
		return "", err
	}
	id, err := s.queries.InsertNotification(ctx, sqlc.InsertNotificationParams{UserID: uid, JobID: jid, Title: title, Body: body})
	if err != nil {
		return "", fmt.Errorf("store.InsertNotification: %w", err)
	}
	return id.String(), nil
}

// Package store is the notify context's Postgres store: the notifications table.
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
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

// ListNotifications returns the user's 50 newest Notifications, unread first.
func (s *Store) ListNotifications(ctx context.Context, userID string) ([]dto.Notification, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListNotifications(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListNotifications: %w", err)
	}
	out := make([]dto.Notification, len(rows))
	for i, r := range rows {
		out[i] = dto.Notification{
			ID:        r.ID.String(),
			JobID:     r.JobID.String(),
			Title:     r.Title,
			Body:      r.Body,
			CreatedAt: r.CreatedAt.Time,
			ReadAt:    data.TimePtr(r.ReadAt),
		}
	}
	return out, nil
}

// MarkNotificationRead sets read_at on the user's own unread Notification; anything else is a no-op.
func (s *Store) MarkNotificationRead(ctx context.Context, userID, notificationID string) error {
	uid, err := data.UUID(userID)
	if err != nil {
		return err
	}
	nid, err := data.UUID(notificationID)
	if err != nil {
		return err
	}
	if err := s.queries.MarkNotificationRead(ctx, sqlc.MarkNotificationReadParams{ID: nid, UserID: uid}); err != nil {
		return fmt.Errorf("store.MarkNotificationRead: %w", err)
	}
	return nil
}

// MarkAllNotificationsRead sets read_at on every unread Notification of the user.
func (s *Store) MarkAllNotificationsRead(ctx context.Context, userID string) error {
	uid, err := data.UUID(userID)
	if err != nil {
		return err
	}
	if err := s.queries.MarkAllNotificationsRead(ctx, uid); err != nil {
		return fmt.Errorf("store.MarkAllNotificationsRead: %w", err)
	}
	return nil
}

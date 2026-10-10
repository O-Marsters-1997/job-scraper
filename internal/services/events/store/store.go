// Package store is the events context's Postgres store: the events table.
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/events/store/sqlc"
)

type Store struct {
	queries *sqlc.Queries
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{queries: sqlc.New(pool)}
}

// InsertEvent appends an event. A non-nil tx joins the caller's transaction;
// nil writes on the pool. An empty subjectID stores NULL.
func (s *Store) InsertEvent(ctx context.Context, tx pgx.Tx, userID, eventType, subjectID string, props []byte) error {
	uid, err := data.UUID(userID)
	if err != nil {
		return err
	}
	var sid pgtype.UUID
	if subjectID != "" {
		if sid, err = data.UUID(subjectID); err != nil {
			return err
		}
	}
	q := s.queries
	if tx != nil {
		q = q.WithTx(tx)
	}
	if err := q.InsertEvent(ctx, sqlc.InsertEventParams{
		UserID: uid, Type: sqlc.EventType(eventType), SubjectID: sid, Props: props,
	}); err != nil {
		return fmt.Errorf("store.InsertEvent: %w", err)
	}
	return nil
}

// ListEvents returns userID's events oldest first; an empty eventType
// returns every type.
func (s *Store) ListEvents(ctx context.Context, userID, eventType string) ([]dto.Event, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, err
	}
	params := sqlc.ListEventsByUserParams{UserID: uid}
	if eventType != "" {
		params.Type = sqlc.NullEventType{EventType: sqlc.EventType(eventType), Valid: true}
	}
	rows, err := s.queries.ListEventsByUser(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("store.ListEvents: %w", err)
	}
	out := make([]dto.Event, len(rows))
	for i, r := range rows {
		out[i] = dto.Event{
			ID: r.ID.String(), Type: string(r.Type), Props: r.Props, CreatedAt: r.CreatedAt.Time,
		}
		if r.SubjectID.Valid {
			out[i].SubjectID = r.SubjectID.String()
		}
	}
	return out, nil
}

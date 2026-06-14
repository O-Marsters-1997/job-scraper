package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

// AddTrackedDoc inserts a tracked doc for the user, ignoring duplicates.
func (db *DB) AddTrackedDoc(ctx context.Context, input dto.AddTrackedDocInput) error {
	const q = `
		INSERT INTO tracked_docs (user_id, doc_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING`

	uid, err := parseUUID(input.UserID)
	if err != nil {
		return err
	}
	if _, err := db.pool.Exec(ctx, q, uid, input.DocID); err != nil {
		return fmt.Errorf("db.AddTrackedDoc: %w", err)
	}
	return nil
}

// RemoveTrackedDoc deletes a tracked doc for the user. Returns ErrTrackedDocNotFound
// if no row was deleted.
func (db *DB) RemoveTrackedDoc(ctx context.Context, userID, docID string) error {
	const q = `DELETE FROM tracked_docs WHERE user_id = $1 AND doc_id = $2`

	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	tag, err := db.pool.Exec(ctx, q, uid, docID)
	if err != nil {
		return fmt.Errorf("db.RemoveTrackedDoc: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return providers.ErrTrackedDocNotFound
	}
	return nil
}

// ListTrackedDocs returns all tracked docs for the user, ordered by added_at.
func (db *DB) ListTrackedDocs(ctx context.Context, userID string) ([]dto.TrackedDoc, error) {
	const q = `
		SELECT id, user_id, doc_id, added_at
		FROM tracked_docs
		WHERE user_id = $1
		ORDER BY added_at`

	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := db.pool.Query(ctx, q, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("db.ListTrackedDocs: %w", err)
	}
	defer rows.Close()

	var out []dto.TrackedDoc
	for rows.Next() {
		var (
			id      pgtype.UUID
			uid     pgtype.UUID
			docID   string
			addedAt pgtype.Timestamptz
		)
		if err := rows.Scan(&id, &uid, &docID, &addedAt); err != nil {
			return nil, fmt.Errorf("db.ListTrackedDocs scan: %w", err)
		}
		out = append(out, dto.TrackedDoc{
			ID:      id.String(),
			UserID:  uid.String(),
			DocID:   docID,
			AddedAt: addedAt.Time,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db.ListTrackedDocs rows: %w", err)
	}
	return out, nil
}

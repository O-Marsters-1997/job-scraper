package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store/sqlc"
)

func (s *Store) SaveOutreachHook(ctx context.Context, userID, contactID, text, model string, cost float64) error {
	uid, err := parseID(userID, ErrContactNotFound)
	if err != nil {
		return err
	}
	cid, err := parseID(contactID, ErrContactNotFound)
	if err != nil {
		return err
	}
	_, err = s.queries.UpsertOutreachHook(ctx, sqlc.UpsertOutreachHookParams{
		ContactID: cid, UserID: uid, Text: text, Model: model, Cost: float32(cost),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrContactNotFound
	}
	if err != nil {
		return fmt.Errorf("store.SaveOutreachHook: %w", err)
	}
	return nil
}

func (s *Store) OutreachHook(ctx context.Context, userID, contactID string) (text string, totalCost float64, err error) {
	uid, err := parseID(userID, ErrContactNotFound)
	if err != nil {
		return "", 0, err
	}
	cid, err := parseID(contactID, ErrContactNotFound)
	if err != nil {
		return "", 0, err
	}
	row, err := s.queries.GetOutreachHook(ctx, sqlc.GetOutreachHookParams{ContactID: cid, UserID: uid})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", 0, nil
	}
	if err != nil {
		return "", 0, fmt.Errorf("store.OutreachHook: %w", err)
	}
	return row.Text, float64(row.Cost), nil
}

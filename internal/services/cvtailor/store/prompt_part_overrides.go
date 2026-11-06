package store

import (
	"context"
	"fmt"

	"github.com/ollymarsters/job-scraper/internal/promptparts"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store/sqlc"
)

func (s *Store) ListPartOverrides(ctx context.Context, userID string, g promptparts.GenerationName) ([]promptparts.Override, error) {
	uid, err := parseID(userID, ErrPositionNotFound)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListPromptPartOverrides(ctx, sqlc.ListPromptPartOverridesParams{UserID: uid, Generation: string(g)})
	if err != nil {
		return nil, fmt.Errorf("store.ListPartOverrides: %w", err)
	}
	out := make([]promptparts.Override, len(rows))
	for i, r := range rows {
		out[i] = promptparts.Override{Path: r.Path, Text: r.Text, BaseDefault: r.BaseDefault}
	}
	return out, nil
}

func (s *Store) SavePartOverride(ctx context.Context, userID string, g promptparts.GenerationName, o promptparts.Override) error {
	uid, err := parseID(userID, ErrPositionNotFound)
	if err != nil {
		return err
	}
	err = s.queries.UpsertPromptPartOverride(ctx, sqlc.UpsertPromptPartOverrideParams{
		UserID: uid, Generation: string(g), Path: o.Path, Text: o.Text, BaseDefault: o.BaseDefault,
	})
	if err != nil {
		return fmt.Errorf("store.SavePartOverride: %w", err)
	}
	return nil
}

func (s *Store) DeletePartOverride(ctx context.Context, userID string, g promptparts.GenerationName, path string) error {
	uid, err := parseID(userID, ErrPositionNotFound)
	if err != nil {
		return err
	}
	err = s.queries.DeletePromptPartOverride(ctx, sqlc.DeletePromptPartOverrideParams{UserID: uid, Generation: string(g), Path: path})
	if err != nil {
		return fmt.Errorf("store.DeletePartOverride: %w", err)
	}
	return nil
}

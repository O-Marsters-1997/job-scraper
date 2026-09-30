// Package store is the cvtemplates context's Postgres store: tracked_docs
// and tracked_doc_tabs.
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates/store/sqlc"
)

var (
	ErrTrackedDocNotFound = apperr.NotFound("tracked doc not found")
	ErrTabNotFound        = apperr.NotFound("tab not found")
)

type Store struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, queries: sqlc.New(pool)}
}

func (s *Store) AddTrackedDoc(ctx context.Context, input dto.AddTrackedDocInput) error {
	uid, err := data.UUID(input.UserID)
	if err != nil {
		return err
	}
	if err := s.queries.AddTrackedDoc(ctx, sqlc.AddTrackedDocParams{UserID: uid, DocID: input.DocID}); err != nil {
		return fmt.Errorf("store.AddTrackedDoc: %w", err)
	}
	return nil
}

func (s *Store) RemoveTrackedDoc(ctx context.Context, userID, docID string) error {
	uid, err := data.UUID(userID)
	if err != nil {
		return err
	}
	n, err := s.queries.RemoveTrackedDoc(ctx, sqlc.RemoveTrackedDocParams{UserID: uid, DocID: docID})
	if err != nil {
		return fmt.Errorf("store.RemoveTrackedDoc: %w", err)
	}
	if n == 0 {
		return ErrTrackedDocNotFound
	}
	return nil
}

func (s *Store) ListTrackedDocs(ctx context.Context, userID string) ([]dto.TrackedDoc, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListTrackedDocs(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListTrackedDocs: %w", err)
	}
	out := make([]dto.TrackedDoc, len(rows))
	for i, row := range rows {
		out[i] = toTrackedDocDTO(row)
	}
	return out, nil
}

func (s *Store) EnsureTabs(ctx context.Context, trackedDocID string, tabIDs, titles []string) error {
	if len(tabIDs) == 0 {
		return nil
	}
	tdID, err := data.UUID(trackedDocID)
	if err != nil {
		return err
	}
	if err := s.queries.EnsureTabs(ctx, sqlc.EnsureTabsParams{
		TrackedDocID: tdID,
		Column2:      tabIDs,
		Column3:      titles,
	}); err != nil {
		return fmt.Errorf("store.EnsureTabs: %w", err)
	}
	return nil
}

func (s *Store) ListTabs(ctx context.Context, trackedDocID string) ([]dto.Tab, error) {
	tdID, err := data.UUID(trackedDocID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListTabs(ctx, tdID)
	if err != nil {
		return nil, fmt.Errorf("store.ListTabs: %w", err)
	}
	out := make([]dto.Tab, len(rows))
	for i, row := range rows {
		out[i] = toTabDTO(row)
	}
	return out, nil
}

func (s *Store) SetTabVisible(ctx context.Context, userID, docID, tabID string, visible bool) error {
	uid, err := data.UUID(userID)
	if err != nil {
		return err
	}
	n, err := s.queries.SetTabVisible(ctx, sqlc.SetTabVisibleParams{UserID: uid, DocID: docID, TabID: tabID, Visible: visible})
	if err != nil {
		return fmt.Errorf("store.SetTabVisible: %w", err)
	}
	if n == 0 {
		return ErrTabNotFound
	}
	return nil
}


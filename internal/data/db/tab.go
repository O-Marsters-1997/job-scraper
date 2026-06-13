package db

import (
	"context"
	"fmt"

	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func fromTab(t pgsqlc.TrackedDocTab) dto.Tab {
	return dto.Tab{
		ID:           t.ID.String(),
		TrackedDocID: t.TrackedDocID.String(),
		TabID:        t.TabID,
		Title:        t.Title,
		Visible:      t.Visible,
		CreatedAt:    t.CreatedAt.Time,
	}
}

func (db *DB) EnsureTabs(ctx context.Context, trackedDocID string, tabIDs []string, titles []string) error {
	if len(tabIDs) == 0 {
		return nil
	}
	tdID, err := parseUUID(trackedDocID)
	if err != nil {
		return err
	}
	if err := db.queries.EnsureTabs(ctx, pgsqlc.EnsureTabsParams{
		TrackedDocID: tdID,
		Column2:      tabIDs,
		Column3:      titles,
	}); err != nil {
		return fmt.Errorf("db.EnsureTabs: %w", err)
	}
	return nil
}

func (db *DB) ListTabs(ctx context.Context, trackedDocID string) ([]dto.Tab, error) {
	tdID, err := parseUUID(trackedDocID)
	if err != nil {
		return nil, err
	}
	rows, err := db.queries.ListTabs(ctx, tdID)
	if err != nil {
		return nil, fmt.Errorf("db.ListTabs: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	out := make([]dto.Tab, len(rows))
	for i, r := range rows {
		out[i] = fromTab(r)
	}
	return out, nil
}

func (db *DB) HideTab(ctx context.Context, userID, docID, tabID string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	n, err := db.queries.HideTab(ctx, pgsqlc.HideTabParams{
		UserID: uid,
		DocID:  docID,
		TabID:  tabID,
	})
	if err != nil {
		return fmt.Errorf("db.HideTab: %w", err)
	}
	if n == 0 {
		return providers.ErrTabNotFound
	}
	return nil
}

func (db *DB) ShowTab(ctx context.Context, userID, docID, tabID string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return err
	}
	n, err := db.queries.ShowTab(ctx, pgsqlc.ShowTabParams{
		UserID: uid,
		DocID:  docID,
		TabID:  tabID,
	})
	if err != nil {
		return fmt.Errorf("db.ShowTab: %w", err)
	}
	if n == 0 {
		return providers.ErrTabNotFound
	}
	return nil
}

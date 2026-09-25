package providers

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

var ErrTabNotFound = apperr.NotFound("tab not found")

type TabProvider interface {
	EnsureTabs(ctx context.Context, trackedDocID string, tabIDs []string, titles []string) error
	ListTabs(ctx context.Context, trackedDocID string) ([]dto.Tab, error)
	HideTab(ctx context.Context, userID, docID, tabID string) error
	ShowTab(ctx context.Context, userID, docID, tabID string) error
}

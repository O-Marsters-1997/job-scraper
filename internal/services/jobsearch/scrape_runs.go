package jobsearch

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func (m *Module) ScrapeRunPageDone(ctx context.Context, runID string) error {
	return m.store.ScrapeRunPageDone(ctx, runID)
}

func (m *Module) ScrapeRunDeferred(ctx context.Context, runID string) error {
	return m.store.ScrapeRunDeferred(ctx, runID)
}

func (m *Module) ScrapeRunAttemptFailed(ctx context.Context, runID string, page, attempt int, message string) error {
	return m.store.ScrapeRunAttemptFailed(ctx, runID, page, attempt, message)
}

func (m *Module) FinishScrapeRun(ctx context.Context, runID string, stop dto.StopReason) error {
	return m.store.FinishScrapeRun(ctx, runID, stop)
}

func (m *Module) DeleteExpiredScrapeRuns(ctx context.Context) error {
	return m.store.DeleteExpiredScrapeRuns(ctx)
}

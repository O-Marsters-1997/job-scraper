package jobsearch

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func (m *Module) SyncWTTJSitemap(ctx context.Context, jobIDs, companyNames []string) (dto.SitemapDiff, error) {
	return m.store.SyncWTTJSitemap(ctx, jobIDs, companyNames)
}

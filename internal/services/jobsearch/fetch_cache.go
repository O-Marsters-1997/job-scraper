package jobsearch

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func (m *Module) LookupFetch(ctx context.Context, url string) (dto.CachedResponse, bool, error) {
	return m.store.LookupFetch(ctx, url)
}

func (m *Module) PutFetch(ctx context.Context, resp dto.CachedResponse) error {
	return m.store.PutFetch(ctx, resp)
}

func (m *Module) ForgetFetches(ctx context.Context, urls []string) error {
	return m.store.ForgetFetches(ctx, urls)
}

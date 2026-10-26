package store

import (
	"context"
	"sync"
	"time"

	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store/sqlc"
)

const coverageTTL = 2 * time.Minute

type coverageCache struct {
	mu       sync.Mutex
	at       time.Time
	all      int
	verified int
}

func (c *coverageCache) get(ctx context.Context, queries *sqlc.Queries) (all, verified int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.at.IsZero() && time.Since(c.at) < coverageTTL {
		return c.all, c.verified, nil
	}
	row, err := queries.CountCompanyBoardCoverage(ctx)
	if err != nil {
		return 0, 0, err
	}
	c.at, c.all, c.verified = time.Now(), int(row.AllCompanies), int(row.VerifiedCompanies)
	return c.all, c.verified, nil
}

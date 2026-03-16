package sources

import (
	"context"
	"time"
)

// Source is the interface every job board scraper must implement.
type Source interface {
	// Name returns the canonical identifier for this source, e.g. "greenhouse".
	Name() string

	// FetchSchedule returns the cron expression that controls how often this
	// source is scraped, e.g. "0 */6 * * *" for every 6 hours.
	FetchSchedule() string

	// FetchURLs retrieves job URLs from a single page of the source.
	FetchURLs(ctx context.Context) ([]string, error)

	// Iterate pages through all job URLs from the source, calling fn for each
	// page's raw URLs. fn returning stop=true triggers early termination (e.g.
	// no new URLs on this page). Respects ctx cancellation.
	Iterate(ctx context.Context, fn func(ctx context.Context, urls []string) (stop bool, err error)) error

	// MinScrapeInterval returns the minimum duration that must have elapsed
	// since the last scrape before this source may be scraped again.
	MinScrapeInterval() time.Duration
}

package sources

import "context"

// URLFilter filters a batch of URLs down to those not yet known.
// Returning an empty slice signals the caller to stop iterating.
type URLFilter func(ctx context.Context, urls []string) ([]string, error)

// Source is the interface every job board scraper must implement.
type Source interface {
	// Name returns the canonical identifier for this source, e.g. "greenhouse".
	Name() string

	// FetchURLs retrieves job URLs from a single page of the source.
	FetchURLs(ctx context.Context) ([]string, error)

	// Iterate retrieves all job URLs across all pages of the source,
	// handling pagination internally. filter is called per page and may
	// trigger early termination when it returns an empty slice.
	Iterate(ctx context.Context, filter URLFilter) ([]string, error)
}

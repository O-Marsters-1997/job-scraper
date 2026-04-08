package providers

import (
	"context"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// JobProvider is the single access point for job persistence.
// Callers never import pgsqlc or pgtype directly.
type JobProvider interface {
	// Save writes one or more jobs, inserting or updating on URL conflict.
	// The implementation selects the most efficient DB path based on input size.
	// Passing an empty slice is a no-op.
	Save(ctx context.Context, jobs []dto.Job) error

	NewURLs(ctx context.Context, urls []string) ([]string, error)

	// List returns all stored jobs ordered by scrape time descending.
	List(ctx context.Context) ([]dto.Job, error)

	// ListSince returns jobs scraped after the given time, ordered by scrape time descending.
	ListSince(ctx context.Context, since time.Time) ([]dto.Job, error)
}

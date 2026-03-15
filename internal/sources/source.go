package sources

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// Source is the interface every job board scraper must implement.
type Source interface {
	// Name returns the canonical identifier for this source, e.g. "greenhouse".
	Name() string

	// FetchJobs retrieves all open job postings from the source.
	FetchJobs(ctx context.Context) ([]dto.Job, error)
}

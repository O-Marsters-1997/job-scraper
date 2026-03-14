package sources

import (
	"context"
	"time"
)

// Job is the normalised representation of a job posting across all sources.
type Job struct {
	ID          int64
	Title       string
	Location    string
	URL         string
	CompanySlug string // board token / slug used to fetch the job
	UpdatedAt   time.Time
}

// Source is the interface every job board scraper must implement.
type Source interface {
	// Name returns the canonical identifier for this source, e.g. "greenhouse".
	Name() string

	// FetchJobs retrieves all open job postings from the source.
	FetchJobs(ctx context.Context) ([]Job, error)
}

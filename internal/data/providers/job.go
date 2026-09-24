package providers

import (
	"context"
	"errors"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

var ErrCanonicalConflict = errors.New("canonical job identity conflict")
var ErrInvalidID = errors.New("invalid ID")

type JobPageOptions struct {
	Limit        int32
	CursorTime   time.Time
	CursorID     string
	Availability string
	CompanyID    string
}

type JobPage struct {
	Items      []dto.Job `json:"items"`
	NextCursor string    `json:"next_cursor"`
}

// JobProvider is the single access point for job persistence.
// Callers never import pgsqlc or pgtype directly.
type JobProvider interface {
	// Save writes one or more jobs, inserting or updating on URL conflict,
	// and returns the saved jobs with their DB-assigned IDs populated.
	// Passing an empty slice is a no-op.
	Save(ctx context.Context, jobs []dto.Job) ([]dto.Job, error)

	NewURLs(ctx context.Context, urls []string) ([]string, error)

	// List returns all stored jobs for the given user, joining scores where available,
	// ordered by suitability score descending then scrape time descending.
	List(ctx context.Context, userID string) ([]dto.Job, error)
	Page(ctx context.Context, userID string, options JobPageOptions) (JobPage, error)

	// GetJob returns a single job with its score for the given user.
	GetJob(ctx context.Context, jobID, userID string) (dto.Job, error)

	// ListSince returns jobs scraped after the given time, ordered by scrape time descending.
	ListSince(ctx context.Context, since time.Time) ([]dto.Job, error)
}

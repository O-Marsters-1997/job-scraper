package providers

import "github.com/ollymarsters/job-scraper/internal/apperr"

// ErrNotFound is the shared not-found sentinel for every legacy provider.
var ErrNotFound = apperr.NotFound("not found")

package data

import "github.com/ollymarsters/job-scraper/internal/apperr"

// ErrNotFound is the shared not-found sentinel for stores with no
// domain-specific detail to add.
var ErrNotFound = apperr.NotFound("not found")

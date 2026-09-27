package data

import "github.com/ollymarsters/job-scraper/internal/apperr"

// ErrNotFound is the not-found sentinel for contexts still on the legacy store.
var ErrNotFound = apperr.NotFound("not found")

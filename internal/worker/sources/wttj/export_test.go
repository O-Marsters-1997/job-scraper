package wttj

import "github.com/ollymarsters/job-scraper/internal/worker/sources"

func ResetLimiter() { limiter = sources.NewGate(0, 0) }

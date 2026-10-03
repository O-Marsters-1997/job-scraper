package wis

import "github.com/ollymarsters/job-scraper/internal/worker/sources"

func StartURL(s Search) string { return s.startURL() }

func ResetLimiter() { limiter = sources.NewGate(0, wisDefaultBlock) }

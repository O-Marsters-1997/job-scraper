package wttj

import "github.com/ollymarsters/job-scraper/internal/worker/sources"

func NewSharedNoGap() *Shared {
	s := NewShared()
	s.gate = sources.NewGate(0, 0)
	return s
}

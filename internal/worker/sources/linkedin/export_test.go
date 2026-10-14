package linkedin

import (
	"time"

	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

func init() { limiter = sources.NewGate(0, 0) }

func SetLimiterGap(gap time.Duration) (restore func()) {
	prev := limiter
	limiter = sources.NewGate(gap, 0)
	return func() { limiter = prev }
}

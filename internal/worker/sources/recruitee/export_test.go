package recruitee

import (
	"time"

	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

func init() { ResetLimiter() }

func ResetLimiter() { limiter = sources.NewGate(0, time.Hour) }

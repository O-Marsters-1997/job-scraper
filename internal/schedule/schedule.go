package schedule

import (
	"context"
	"log/slog"
	"time"

	"github.com/ollymarsters/job-scraper/internal/logger"
)

// Every runs fn immediately, then once per d, logging any error under name.
// It returns when ctx is cancelled.
func Every(ctx context.Context, name string, d time.Duration, fn func(context.Context) error) {
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	for {
		if err := fn(ctx); err != nil {
			slog.ErrorContext(ctx, name+" failed", slog.Any(logger.KeyErr, err)) //nolint:sloglint // pedantic: msg is built from the chore name
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

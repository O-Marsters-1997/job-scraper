package worker

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/ollymarsters/job-scraper/internal/queue"
)

// HandlerFunc processes a single URL.
type HandlerFunc func(ctx context.Context, url string) error

// Run processes one job at a time with a random 10–15s pause between items.
// When the queue is empty it backs off for 15 minutes. Respects ctx cancellation.
func Run(ctx context.Context, q *queue.Queue, handler HandlerFunc) error {
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		url, ok, err := q.Dequeue(ctx)
		if err != nil {
			slog.Error("worker dequeue failed", slog.Any("err", err))
			if !sleep(ctx, 5*time.Second) {
				return nil
			}
			continue
		}

		if !ok {
			// Queue empty — back off for 15 minutes.
			slog.Info("worker: queue empty, sleeping", slog.Duration("for", 15*time.Minute))
			if !sleep(ctx, 15*time.Minute) {
				return nil
			}
			continue
		}

		if err := handler(ctx, url); err != nil {
			slog.Error("worker handler failed", slog.String("url", url), slog.Any("err", err))
		} else {
			slog.Info("worker processed", slog.String("url", url))
		}

		wait := 10*time.Second + time.Duration(rand.Int64N(int64(5*time.Second)))
		if !sleep(ctx, wait) {
			return nil
		}
	}
}

// sleep blocks for d, returning false if ctx is cancelled first.
func sleep(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

package worker

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/ollymarsters/job-scraper/internal/queue"
)

type HandlerFunc func(ctx context.Context, url string) error

// Run processes one job at a time with a random 10–15s pause between items.
// When the queue is empty it backs off for 15 minutes. Respects ctx cancellation.
func Run(ctx context.Context, q queue.JobQueue, handler HandlerFunc) error {
	w := newWorker()
	return w.run(ctx, q, handler)
}

// worker holds timing configuration, allowing tests to inject zero-duration sleeps.
type worker struct {
	itemDelay  func() time.Duration
	emptyDelay time.Duration
	errDelay   time.Duration
}

func newWorker() worker {
	return worker{
		itemDelay:  func() time.Duration { return 10*time.Second + time.Duration(rand.Int64N(int64(5*time.Second))) },
		emptyDelay: 15 * time.Minute,
		errDelay:   5 * time.Second,
	}
}

func (w *worker) run(ctx context.Context, q queue.JobQueue, handler HandlerFunc) error {
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		url, ok, err := q.Dequeue(ctx)
		if err != nil {
			slog.Error("worker dequeue failed", slog.Any("err", err))
			if !sleep(ctx, w.errDelay) {
				return nil
			}
			continue
		}

		if !ok {
			slog.Info("worker: queue empty, sleeping", slog.Duration("for", w.emptyDelay))
			if !sleep(ctx, w.emptyDelay) {
				return nil
			}
			continue
		}

		if err := handler(ctx, url); err != nil {
			slog.Error("worker handler failed", slog.String("url", url), slog.Any("err", err))
		} else {
			slog.Info("worker processed", slog.String("url", url))
		}

		if !sleep(ctx, w.itemDelay()) {
			return nil
		}
	}
}

// sleep blocks for d, returning false if ctx is cancelled first.
func sleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

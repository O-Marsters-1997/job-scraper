package worker

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
)

type HandlerFunc func(ctx context.Context, job dto.QueuedJob) error

// Run processes one job at a time with a random 10–15s pause between items.
// When the queue is empty it backs off 30s. Respects ctx cancellation.
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
		emptyDelay: 30 * time.Second,
		errDelay:   5 * time.Second,
	}
}

func (w *worker) run(ctx context.Context, q queue.JobQueue, handler HandlerFunc) error {
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		job, ok, err := q.Dequeue(ctx)
		if err != nil {
			slog.Error("dequeue failed", slog.Any("err", err))
			if !sleep(ctx, w.errDelay) {
				return nil
			}
			continue
		}

		if !ok {
			slog.Info("queue empty, sleeping", slog.Duration("for", w.emptyDelay))
			if !sleep(ctx, w.emptyDelay) {
				return nil
			}
			continue
		}

		if err := handler(ctx, job); err != nil {
			slog.Error("handler failed", slog.String("url", job.URL), slog.Any("err", err))
			if nackErr := q.Nack(ctx, job.URL); nackErr != nil {
				slog.Error("nack failed", slog.String("url", job.URL), slog.Any("err", nackErr))
			}
		} else {
			slog.Info("processed", slog.String("url", job.URL))
			if clearErr := q.ClearAttempts(ctx, job.URL); clearErr != nil {
				slog.Error("clear attempts failed", slog.String("url", job.URL), slog.Any("err", clearErr))
			}
		}

		if !sleep(ctx, w.itemDelay()) {
			return nil
		}
	}
}

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

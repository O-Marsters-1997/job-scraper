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

// ScrapeHandlerFunc handles a single on-demand scrape request.
type ScrapeHandlerFunc func(ctx context.Context, req dto.ScrapeRequest) error

// RunScrapeRequests polls the scrape-request queue and calls handler for each
// entry. When the queue is empty it backs off emptyDelay. Respects ctx cancellation.
// It mirrors Run's structure but is simpler: no Nack/ClearAttempts logic needed
// since scrape-now failures are best-effort (the regular schedule will cover it).
func RunScrapeRequests(ctx context.Context, q queue.JobQueue, handler ScrapeHandlerFunc) {
	const emptyDelay = 5 * time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		req, ok, err := q.DequeueScrapeRequest(ctx)
		if err != nil {
			slog.Error("dequeue scrape request failed", slog.Any("err", err))
			if !sleep(ctx, emptyDelay) {
				return
			}
			continue
		}
		if !ok {
			if !sleep(ctx, emptyDelay) {
				return
			}
			continue
		}
		slog.Info("scrape request received", slog.String("source", req.Target.Source), slog.String("value", req.Target.Value))
		if err := handler(ctx, req); err != nil {
			slog.Error("scrape request handler failed",
				slog.String("source", req.Target.Source),
				slog.String("value", req.Target.Value),
				slog.Any("err", err),
			)
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

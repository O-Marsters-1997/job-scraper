package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
)

type HandlerFunc func(ctx context.Context, job dto.QueuedJob) error

func Run(ctx context.Context, q queue.JobQueue, handler HandlerFunc) error {
	w := newWorker()
	return w.run(ctx, q, handler)
}

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

		item, ok, err := q.ClaimReady(ctx, queue.Detail, 5*time.Minute)
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
		var job dto.QueuedJob
		if err := json.Unmarshal(item.Payload, &job); err != nil {
			_ = q.Nack(ctx, item, err.Error())
			continue
		}

		if err := handler(ctx, job); err != nil {
			slog.Error("handler failed", slog.String("url", job.URL), slog.Any("err", err))
			if nackErr := q.Nack(context.WithoutCancel(ctx), item, err.Error()); nackErr != nil {
				slog.Error("nack failed", slog.String("url", job.URL), slog.Any("err", nackErr))
			}
		} else {
			slog.Info("processed", slog.String("url", job.URL))
			if ackErr := q.Ack(context.WithoutCancel(ctx), item); ackErr != nil {
				slog.Error("ack failed", slog.String("url", job.URL), slog.Any("err", ackErr))
			}
		}

		if !sleep(ctx, w.itemDelay()) {
			return nil
		}
	}
}

type ScrapeHandlerFunc func(ctx context.Context, req dto.ScrapeRequest) error

func RunScrapeRequests(ctx context.Context, q queue.JobQueue, handler ScrapeHandlerFunc) {
	const emptyDelay = 5 * time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		item, ok, err := q.ClaimReady(ctx, queue.ScrapeRequest, 30*time.Minute)
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
		var req dto.ScrapeRequest
		if err := json.Unmarshal(item.Payload, &req); err != nil {
			_ = q.Nack(ctx, item, err.Error())
			continue
		}
		slog.Info("scrape request received", slog.String("source", req.Target.Source), slog.String("value", req.Target.Value))
		if err := handler(ctx, req); err != nil {
			slog.Error("scrape request handler failed",
				slog.String("source", req.Target.Source),
				slog.String("value", req.Target.Value),
				slog.Any("err", err),
			)
			if nackErr := q.Nack(context.WithoutCancel(ctx), item, err.Error()); nackErr != nil {
				slog.Error("nack scrape request failed", slog.Any("err", nackErr))
			}
		} else if ackErr := q.Ack(context.WithoutCancel(ctx), item); ackErr != nil {
			slog.Error("ack scrape request failed", slog.Any("err", ackErr))
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

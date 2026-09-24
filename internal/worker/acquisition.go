package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ollymarsters/job-scraper/internal/queue"
)

type SourceQueue interface {
	ClaimSource(context.Context, time.Duration) (queue.SourceItem, bool, error)
	AckSource(context.Context, queue.SourceItem) error
	NackSource(context.Context, queue.SourceItem, string) error
	RenewSource(context.Context, queue.SourceItem, time.Duration) error
}

type AcquisitionHandler func(context.Context, queue.SourceItem) error

func RunAcquisition(ctx context.Context, q SourceQueue, size int, handler AcquisitionHandler) {
	if size < 1 {
		return
	}
	var workers sync.WaitGroup
	for range size {
		workers.Go(func() {
			for ctx.Err() == nil {
				item, ok, err := q.ClaimSource(ctx, time.Minute)
				if err != nil {
					slog.Error("source claim failed", slog.Any("err", err))
				}
				if !ok || err != nil {
					if !sleep(ctx, 100*time.Millisecond) {
						return
					}
					continue
				}
				runAcquisition(ctx, q, item, handler)
			}
		})
	}
	workers.Wait()
}

func runAcquisition(ctx context.Context, q SourceQueue, item queue.SourceItem, handler AcquisitionHandler) {
	workCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	stopped := make(chan struct{})
	lost := make(chan error, 1)
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-workCtx.Done():
				return
			case <-ticker.C:
				if err := q.RenewSource(workCtx, item, time.Minute); err != nil {
					lost <- err
					cancel()
					return
				}
			}
		}
	}()
	err := handler(workCtx, item)
	cancel()
	<-stopped
	select {
	case leaseErr := <-lost:
		slog.Error("source lease lost", slog.String("source", item.Source), slog.Any("err", leaseErr))
		return
	default:
	}
	transitionCtx, stop := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer stop()
	if err != nil {
		if nackErr := q.NackSource(transitionCtx, item, err.Error()); nackErr != nil {
			slog.Error("source nack failed", slog.String("source", item.Source), slog.Any("err", nackErr))
		}
		return
	}
	if ackErr := q.AckSource(transitionCtx, item); ackErr != nil {
		slog.Error("source ack failed", slog.String("source", item.Source), slog.Any("err", ackErr))
	}
}

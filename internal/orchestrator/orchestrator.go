package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

// Orchestrator combines the startup seed and recurring cron schedule into a
// single Start/Stop lifecycle. Callers no longer manage a *cron.Cron directly.
type Orchestrator struct {
	srcs []sources.Source
	db   providers.JobProvider
	q    queue.JobQueue
	cr   *cron.Cron
	wg   sync.WaitGroup
}

// New wires together a scrape orchestrator. Dependencies are provided at
// construction time; the context governing goroutine lifetimes is passed to Start.
func New(srcs []sources.Source, db providers.JobProvider, q queue.JobQueue) *Orchestrator {
	return &Orchestrator{srcs: srcs, db: db, q: q}
}

// Start seeds all sources immediately (background, guarded by MinScrapeInterval)
// and registers + starts the cron schedule for recurring scrapes.
// Returns an error only if a cron expression is invalid.
// Non-blocking — returns as soon as the cron scheduler is running.
func (o *Orchestrator) Start(ctx context.Context) error {
	// Seed all sources concurrently in the background.
	go func() {
		var wg sync.WaitGroup
		for _, src := range o.srcs {
			wg.Add(1)
			go func(src sources.Source) {
				defer wg.Done()
				o.runIfReady(ctx, src)
			}(src)
		}
		wg.Wait()
	}()

	o.cr = cron.New()

	for _, src := range o.srcs {
		cfg := src.Cfg()
		if _, err := o.cr.AddFunc(cfg.Schedule, func() {
			slog.Info("cron: starting scrape", slog.String("source", cfg.Name))
			o.wg.Add(1)
			defer o.wg.Done()
			o.runIfReady(ctx, src)
		}); err != nil {
			o.cr.Stop()
			return fmt.Errorf("schedule %s (%s): %w", cfg.Name, cfg.Schedule, err)
		}
		slog.Info("cron: scheduled", slog.String("source", cfg.Name), slog.String("schedule", cfg.Schedule))
	}

	o.cr.Start()
	return nil
}

// Stop halts the cron scheduler and waits for any in-flight cron-triggered
// scrape to finish. Goroutine cancellation is the caller's responsibility via ctx.
// Safe to call multiple times.
func (o *Orchestrator) Stop() {
	if o.cr != nil {
		o.cr.Stop()
	}
	o.wg.Wait()
}

func (o *Orchestrator) runIfReady(ctx context.Context, src sources.Source) {
	cfg := src.Cfg()
	last, ok, err := o.q.GetLastScraped(ctx, cfg.Name)
	if err != nil {
		slog.Error("could not read last scraped", slog.String("source", cfg.Name), slog.Any("err", err))
		// fail open — proceed with the scrape
	}
	if ok && time.Since(last) < cfg.MinScrapeInterval {
		slog.Info("skipping scrape: ran recently",
			slog.String("source", cfg.Name),
			slog.Duration("ago", time.Since(last)),
		)
		return
	}
	if err := o.run(ctx, src); err != nil {
		slog.Error("scrape failed", slog.String("source", cfg.Name), slog.Any("err", err))
		return
	}
	if err := o.q.SetLastScraped(ctx, cfg.Name); err != nil {
		slog.Error("could not set last scraped", slog.String("source", cfg.Name), slog.Any("err", err))
	}
}

func (o *Orchestrator) run(ctx context.Context, src sources.Source) error {
	name := src.Cfg().Name
	seen := make(map[string]struct{})

	return src.Iterate(ctx, func(ctx context.Context, rawURLs []string) (bool, error) {
		newURLs, err := o.db.NewURLs(ctx, rawURLs)
		if err != nil {
			slog.Error("filter failed", slog.String("source", name), slog.Any("err", err))
			newURLs = rawURLs // fail open
		}

		deduped := make([]string, 0, len(newURLs))
		for _, u := range newURLs {
			if u == "" {
				continue
			}
			if _, ok := seen[u]; !ok {
				seen[u] = struct{}{}
				deduped = append(deduped, u)
			}
		}

		if len(deduped) == 0 {
			return true, nil // nothing new — stop iterating
		}

		if err := o.q.Enqueue(ctx, deduped); err != nil {
			return false, err
		}

		slog.Info("enqueued page", slog.String("source", name), slog.Int("count", len(deduped)))
		return false, nil
	})
}

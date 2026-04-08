package scraper

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

type Orchestrator struct {
	srcs []sources.Source
	db   providers.JobProvider
	q    queue.JobQueue
	cr   *cron.Cron
	wg   sync.WaitGroup
}

func New(srcs []sources.Source, db providers.JobProvider, q queue.JobQueue) *Orchestrator {
	return &Orchestrator{srcs: srcs, db: db, q: q}
}

func (o *Orchestrator) Start(ctx context.Context) error {
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
			slog.Info("cron: starting scrape",
				slog.String("source", cfg.Name),
			)
			o.wg.Add(1)
			defer o.wg.Done()
			o.runIfReady(ctx, src)
		}); err != nil {
			o.cr.Stop()
			return fmt.Errorf("schedule %s (%s): %w", cfg.Name, cfg.Schedule, err)
		}
		slog.Info("cron: scheduled",
			slog.String("source", cfg.Name),
			slog.String("schedule", cfg.Schedule),
		)
	}

	o.cr.Start()
	return nil
}

func (o *Orchestrator) Stop() {
	if o.cr != nil {
		o.cr.Stop()
	}
	o.wg.Wait()
}

func (o *Orchestrator) runIfReady(ctx context.Context, src sources.Source) {
	cfg := src.Cfg()
	log := slog.With(slog.String("source", cfg.Name))

	last, ok, err := o.q.GetLastScraped(ctx, cfg.Name)
	if err != nil {
		log.Warn("could not read last scraped, proceeding",
			slog.Any("err", err),
		)
	}
	if ok && time.Since(last) < cfg.MinScrapeInterval {
		log.Info("skipping scrape: ran recently", slog.Duration("ago", time.Since(last)))
		return
	}
	if err := o.run(ctx, src); err != nil {
		log.Error("scrape failed", slog.Any("err", err))
		return
	}
	if err := o.q.SetLastScraped(ctx, cfg.Name); err != nil {
		log.Error("could not set last scraped", slog.Any("err", err))
	}
}

func (o *Orchestrator) run(ctx context.Context, src sources.Source) error {
	name := src.Cfg().Name
	log := slog.With(slog.String("source", name))
	seen := make(map[string]struct{})

	return src.Iterate(ctx, func(ctx context.Context, rawURLs []string) (bool, error) {
		newURLs, err := o.db.NewURLs(ctx, rawURLs)
		if err != nil {
			log.Warn("filter failed, using all URLs", slog.Any("err", err))
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
			return true, nil
		}

		if err := o.q.Enqueue(ctx, deduped); err != nil {
			return false, err
		}

		log.Info("enqueued page", slog.Int("count", len(deduped)))
		return false, nil
	})
}

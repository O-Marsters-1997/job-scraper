package scraper

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/queue"

	"github.com/ollymarsters/job-scraper/internal/sources"
)

// Seed runs a startup scrape for every source concurrently.
// It returns immediately; scrapes happen in the background.
func Seed(ctx context.Context, srcs []sources.Source, db providers.JobProvider, q *queue.Queue) {
	go func() {
		var wg sync.WaitGroup
		for _, src := range srcs {
			wg.Add(1)
			go func(src sources.Source) {
				defer wg.Done()
				RunIfReady(ctx, src, db, q)
			}(src)
		}
		wg.Wait()
	}()
}

func RunIfReady(ctx context.Context, src sources.Source, db providers.JobProvider, q *queue.Queue) {
	last, ok, err := q.GetLastScraped(ctx, src.Name())
	if err != nil {
		slog.Error("could not read last scraped", slog.String("source", src.Name()), slog.Any("err", err))
		// fail open — proceed with the scrape
	}
	if ok && time.Since(last) < src.MinScrapeInterval() {
		slog.Info("skipping scrape: ran recently",
			slog.String("source", src.Name()),
			slog.Duration("ago", time.Since(last)),
		)
		return
	}
	if err := run(ctx, src, db, q); err != nil {
		slog.Error("scrape failed", slog.String("source", src.Name()), slog.Any("err", err))
		return
	}
	if err := q.SetLastScraped(ctx, src.Name(), time.Now()); err != nil {
		slog.Error("could not set last scraped", slog.String("source", src.Name()), slog.Any("err", err))
	}
}

func run(ctx context.Context, src sources.Source, db providers.JobProvider, q *queue.Queue) error {
	seen := make(map[string]struct{})

	return src.Iterate(ctx, func(ctx context.Context, rawURLs []string) (bool, error) {
		newURLs, err := db.FilterNewURLs(ctx, rawURLs)
		if err != nil {
			slog.Error("filter failed", slog.String("source", src.Name()), slog.Any("err", err))
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

		if err := q.Enqueue(ctx, deduped, time.Now()); err != nil {
			return false, err
		}

		slog.Info("enqueued page", slog.String("source", src.Name()), slog.Int("count", len(deduped)))
		return false, nil
	})
}

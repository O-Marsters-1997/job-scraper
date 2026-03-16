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

// Run iterates all sources concurrently, deduplicates job URLs, and enqueues
// them for processing. Source errors are logged but do not abort other sources.
func Run(ctx context.Context, srcs []sources.Source, db providers.JobProvider, q *queue.Queue) error {
	filter := sources.URLFilter(db.FilterNewURLs)

	ch := make(chan []string, len(srcs))

	var wg sync.WaitGroup
	for _, src := range srcs {
		wg.Add(1)
		go func(src sources.Source) {
			defer wg.Done()
			urls, err := src.Iterate(ctx, filter)
			if err != nil {
				slog.Error("source iterate failed", slog.String("source", src.Name()), slog.Any("err", err))
				return
			}
			slog.Info("source complete", slog.String("source", src.Name()), slog.Int("count", len(urls)))
			ch <- urls
		}(src)
	}

	wg.Wait()
	close(ch)

	seen := make(map[string]struct{})
	var urls []string
	for batch := range ch {
		for _, u := range batch {
			if u == "" {
				continue
			}
			if _, ok := seen[u]; ok {
				continue
			}
			seen[u] = struct{}{}
			urls = append(urls, u)
		}
	}

	slog.Info("deduplication complete", slog.Int("unique", len(urls)))

	if err := q.Enqueue(ctx, urls, time.Now()); err != nil {
		return err
	}

	slog.Info("enqueue complete", slog.Int("count", len(urls)))
	return nil
}

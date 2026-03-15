package scraper

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

// Run iterates all sources concurrently, deduplicates job URLs, and enqueues
// them for processing. Source errors are logged but do not abort other sources.
func Run(ctx context.Context, srcs []sources.Source, q *queue.Queue) error {
	type result struct {
		name string
		jobs []dto.Job
	}

	ch := make(chan result, len(srcs))

	var wg sync.WaitGroup
	for _, src := range srcs {
		wg.Add(1)
		go func(src sources.Source) {
			defer wg.Done()
			jobs, err := src.Iterate(ctx)
			if err != nil {
				slog.Error("source iterate failed", slog.String("source", src.Name()), slog.Any("err", err))
				return
			}
			slog.Info("source complete", slog.String("source", src.Name()), slog.Int("count", len(jobs)))
			ch <- result{name: src.Name(), jobs: jobs}
		}(src)
	}

	wg.Wait()
	close(ch)

	seen := make(map[string]struct{})
	var urls []string
	for r := range ch {
		for _, j := range r.jobs {
			if j.URL == "" {
				continue
			}
			if _, ok := seen[j.URL]; ok {
				continue
			}
			seen[j.URL] = struct{}{}
			urls = append(urls, j.URL)
		}
	}

	slog.Info("deduplication complete", slog.Int("unique", len(urls)))

	if err := q.Enqueue(ctx, urls, time.Now()); err != nil {
		return err
	}

	slog.Info("enqueue complete", slog.Int("count", len(urls)))
	return nil
}

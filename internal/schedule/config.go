package schedule

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/robfig/cron/v3"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/scraper"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

// Config holds the set of sources to schedule.
type Config struct {
	sources []sources.Source
}

// New constructs a Config from the given sources.
func New(srcs []sources.Source) *Config {
	return &Config{sources: srcs}
}

// Initialize runs a startup scrape for each source (in parallel, guarded by
// runIfReady), registers one cron job per source, starts the scheduler, and
// returns the running *cron.Cron. The caller is responsible for calling Stop()
// on shutdown.
func (c *Config) Initialize(ctx context.Context, db providers.JobProvider, q *queue.Queue) (*cron.Cron, error) {
	cr := cron.New()

	for _, src := range c.sources {
		if _, err := cr.AddFunc(src.FetchSchedule(), func() {
			slog.Info("cron: starting scrape", slog.String("source", src.Name()))
			scraper.RunIfReady(ctx, src, db, q)
		}); err != nil {
			cr.Stop()
			return nil, fmt.Errorf("schedule %s (%s): %w", src.Name(), src.FetchSchedule(), err)
		}
		slog.Info("cron: scheduled", slog.String("source", src.Name()), slog.String("schedule", src.FetchSchedule()))
	}

	cr.Start()
	return cr, nil
}

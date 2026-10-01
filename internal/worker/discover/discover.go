// Package discover turns code-registered Harvesters into board_discover tasks
// on a per-harvester cadence, gating each harvester to one run per interval.
package discover

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
)

// Board is an ATS board a harvester found, identified by source and token.
type Board struct {
	Source, Token string
}

// Harvest is what one harvester run found. Skipped counts candidates that
// resolved to no board or an unsupported source.
type Harvest struct {
	Boards  []Board
	Skipped int
}

// Get fetches url with client and returns the body of a 200 response. A non-empty
// userAgent is sent as the User-Agent header.
func Get(ctx context.Context, client *http.Client, url, userAgent string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: status %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// Harvester yields ATS boards from one external source.
type Harvester interface {
	// Name identifies the harvester in logs and as its gate key ("harvest:" + Name()).
	Name() string
	// Interval is the minimum time between successful runs.
	Interval() time.Duration
	Harvest(ctx context.Context) (Harvest, error)
}

// ScrapeGate persists each harvester's successful run time.
type ScrapeGate interface {
	SetLastScraped(ctx context.Context, source string) error
	GetLastScraped(ctx context.Context, source string) (time.Time, bool, error)
}

// Publisher is the subset of the queue broker the Runner needs.
type Publisher interface {
	Publish(ctx context.Context, task queue.Task) error
}

const gateKeyPrefix = "harvest:"

// Runner publishes a board_discover task per harvested board.
type Runner struct {
	harvesters []Harvester
	publisher  Publisher
	gate       ScrapeGate
}

func NewRunner(hs []Harvester, publisher Publisher, gate ScrapeGate) *Runner {
	return &Runner{harvesters: hs, publisher: publisher, gate: gate}
}

// RunOnce harvests every due harvester and joins their failures.
func (r *Runner) RunOnce(ctx context.Context) error {
	var errs []error
	for _, h := range r.harvesters {
		if err := r.runIfDue(ctx, h); err != nil {
			errs = append(errs, fmt.Errorf("harvest %s: %w", h.Name(), err))
		}
	}
	return errors.Join(errs...)
}

func (r *Runner) runIfDue(ctx context.Context, h Harvester) error {
	key := gateKeyPrefix + h.Name()
	log := slog.With(slog.String("harvester", h.Name()))

	last, ok, err := r.gate.GetLastScraped(ctx, key)
	if err != nil {
		log.WarnContext(ctx, "could not read last harvested, proceeding", slog.Any(logger.KeyErr, err))
	}
	if ok && time.Since(last) < h.Interval() {
		log.InfoContext(ctx, "skipping harvest: ran recently", slog.Duration("ago", time.Since(last)))
		return nil
	}

	log.InfoContext(ctx, "harvest: running")
	found, err := h.Harvest(ctx)
	if err != nil {
		return err
	}

	published := 0
	for _, b := range found.Boards {
		task := queue.Task{Version: 1, ID: uuid.NewString(), Source: b.Source, Kind: queue.BoardDiscoverTask, BoardToken: b.Token}
		if err := r.publisher.Publish(ctx, task); err != nil {
			log.ErrorContext(ctx, "publish failed", slog.String(logger.KeySource, b.Source), slog.String("token", b.Token), slog.Any(logger.KeyErr, err))
			continue
		}
		published++
	}
	log.InfoContext(ctx, "harvest: completed",
		slog.Int(logger.KeyCount, len(found.Boards)), slog.Int("published", published), slog.Int("skipped", found.Skipped))

	if err := r.gate.SetLastScraped(ctx, key); err != nil {
		log.ErrorContext(ctx, "could not set last harvested", slog.Any(logger.KeyErr, err))
	}
	return nil
}

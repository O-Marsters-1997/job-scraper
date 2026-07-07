package sources

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// BoardSpec describes one ATS board integration. URL builds the API endpoint for a
// board token; Parse decodes the response body into jobs. Schedule and
// MinScrapeInterval come from NewBase defaults; set UseProxy only when the board needs
// the Web Unlocker.
type BoardSpec struct {
	Name      string
	URLPrefix string
	UseProxy  bool
	URL       func(token string) string
	Parse     func(body []byte, token string) ([]dto.Job, error)
}

// BoardSource is the shared implementation for ATS "board" sources whose only
// per-source variation is the endpoint URL, the decoder, and the field mapping — all
// captured by BoardSpec.
type BoardSource struct {
	PaginatedBase
	boards []string
	spec   BoardSpec
	done   func(ctx context.Context, token string, urls []string)
}

var _ Source = (*BoardSource)(nil)

func NewBoardSource(boards []string, spec BoardSpec) *BoardSource {
	return &BoardSource{
		PaginatedBase: NewBase(Config{Name: spec.Name, URLPrefix: spec.URLPrefix, UseProxy: spec.UseProxy}),
		boards:        boards,
		spec:          spec,
	}
}

// WithDone registers a hook called after each board token is successfully
// fetched and parsed, letting the caller record per-token freshness (e.g.
// last_checked_at) independent of one-off manual scrapes. urls is every job
// URL the board API returned for that token this run, pre-relevance-gate —
// callers can diff it against previously-known URLs to detect closures.
func (b *BoardSource) WithDone(fn func(ctx context.Context, token string, urls []string)) *BoardSource {
	b.done = fn
	return b
}

// Iterate fetches every configured board token, logging and continuing past
// per-token failures so one dead board doesn't starve the rest of the ATS's
// boards. Errors are joined and returned once all tokens have been attempted.
func (b *BoardSource) Iterate(ctx context.Context, fn func(context.Context, []dto.Job) (bool, error)) error {
	var errs []error
	for _, token := range b.boards {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		body, err := b.Get(ctx, b.spec.URL(token))
		if err != nil {
			slog.Warn("board fetch failed", slog.String("source", b.spec.Name), slog.String("board", token), slog.Any("err", err))
			errs = append(errs, fmt.Errorf("%s: board %s: %w", b.spec.Name, token, err))
			continue
		}
		jobs, err := b.spec.Parse(body, token)
		if err != nil {
			slog.Warn("board parse failed", slog.String("source", b.spec.Name), slog.String("board", token), slog.Any("err", err))
			errs = append(errs, fmt.Errorf("%s: board %s: %w", b.spec.Name, token, err))
			continue
		}
		stop, err := fn(ctx, jobs)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: board %s: %w", b.spec.Name, token, err))
			continue
		}
		if b.done != nil {
			urls := make([]string, len(jobs))
			for i, job := range jobs {
				urls[i] = job.URL
			}
			b.done(ctx, token, urls)
		}
		if stop {
			break
		}
	}
	return errors.Join(errs...)
}

// RFC3339OrNow parses an RFC3339 timestamp, falling back to the current UTC time when
// raw is empty or unparseable.
func RFC3339OrNow(raw string) time.Time {
	if raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			return t
		}
	}
	return time.Now().UTC()
}

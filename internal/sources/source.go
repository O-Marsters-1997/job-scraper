package sources

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	DefaultSchedule          = "0 */6 * * *"
	DefaultMinScrapeInterval = 5 * time.Hour
	DefaultTimeout           = 15 * time.Second
	defaultMinWait           = 2 * time.Second
	defaultMaxWait           = 7 * time.Second
	userAgent                = "Mozilla/5.0 (compatible; job-scraper/1.0)"
)

type Config struct {
	// Name is the canonical identifier, e.g. "wis", "greenhouse".
	Name string

	// Schedule is a cron expression controlling how often this source is polled.
	// Defaults to DefaultSchedule if empty.
	Schedule string

	// MinScrapeInterval is the minimum duration between scrapes.
	// Defaults to DefaultMinScrapeInterval if zero.
	MinScrapeInterval time.Duration

	// URLPrefix is used by PaginatedBase.CanHandle to claim URLs by prefix.
	URLPrefix string
}

type Source interface {
	Cfg() Config

	// CanHandle reports whether this source produced url and can parse its
	// detail page. Used by Dispatch to route dequeued URLs to the right source.
	CanHandle(url string) bool

	// Iterate pages through all job URLs from the source, calling fn for each
	// page's raw URLs. fn returning stop=true triggers early termination.
	// Respects ctx cancellation.
	Iterate(ctx context.Context, fn func(ctx context.Context, urls []string) (stop bool, err error)) error

	// GetDetails fetches url and returns a fully-populated Job.
	GetDetails(ctx context.Context, url string) (dto.Job, error)
}

type PaginatedBase struct {
	cfg    Config
	client *http.Client
}

func NewBase(cfg Config) PaginatedBase {
	if cfg.Schedule == "" {
		cfg.Schedule = DefaultSchedule
	}
	if cfg.MinScrapeInterval == 0 {
		cfg.MinScrapeInterval = DefaultMinScrapeInterval
	}
	return PaginatedBase{
		cfg:    cfg,
		client: &http.Client{Timeout: DefaultTimeout},
	}
}

func (b *PaginatedBase) Cfg() Config { return b.cfg }

func (b *PaginatedBase) CanHandle(url string) bool {
	return strings.HasPrefix(url, b.cfg.URLPrefix)
}

func (b *PaginatedBase) Client() *http.Client { return b.client }

func (b *PaginatedBase) Get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return body, nil
}

func (b *PaginatedBase) IteratePages(
	ctx context.Context,
	fn func(context.Context, []string) (bool, error),
	fetchPage func(context.Context, int) ([]string, int, error),
	resultsPerPage int,
) error {
	page1, total, err := fetchPage(ctx, 1)
	if err != nil {
		return fmt.Errorf("%s page 1: %w", b.cfg.Name, err)
	}

	pages := (total + resultsPerPage - 1) / resultsPerPage
	slog.Info("iterating source", slog.String("source", b.cfg.Name), slog.Int("total", total), slog.Int("pages", pages))

	stop, err := fn(ctx, page1)
	if err != nil || stop {
		return err
	}

	for p := 2; p <= pages; p++ {
		wait := defaultMinWait + time.Duration(rand.Int64N(int64(defaultMaxWait-defaultMinWait)))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}

		slog.Debug("fetching page", slog.String("source", b.cfg.Name), slog.Int("page", p), slog.Int("of", pages))

		pageURLs, _, err := fetchPage(ctx, p)
		if err != nil {
			slog.Error("page failed", slog.String("source", b.cfg.Name), slog.Int("page", p), slog.Any("err", err))
			continue
		}

		stop, err = fn(ctx, pageURLs)
		if err != nil {
			return err
		}
		if stop {
			slog.Info("early stop", slog.String("source", b.cfg.Name), slog.Int("page", p))
			break
		}
	}

	return nil
}

// Dispatch routes url to the first source that claims it via CanHandle and
// calls its GetDetails. Returns an error if no source claims the URL.
func Dispatch(ctx context.Context, srcs []Source, url string) (dto.Job, error) {
	for _, src := range srcs {
		if src.CanHandle(url) {
			return src.GetDetails(ctx, url)
		}
	}
	return dto.Job{}, fmt.Errorf("sources: no handler for %s", url)
}

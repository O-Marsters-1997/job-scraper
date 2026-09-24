package sources

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/proxy"
)

const (
	DefaultSchedule = "0 * * * *"
	DefaultTimeout  = 15 * time.Second
	defaultMinWait  = 2 * time.Second
	defaultMaxWait  = 7 * time.Second
	userAgent       = "Mozilla/5.0 (compatible; job-scraper/1.0)"
)

type Config struct {
	Name     string
	UseProxy bool
}

type Source interface {
	Cfg() Config

	// Iterate pages through all jobs from the source, calling fn for each
	// page's jobs. ATS sources yield fully-populated dto.Job; HTML sources
	// yield partial dto.Job{URL: url}. fn returning stop=true triggers early
	// termination. Respects ctx cancellation.
	Iterate(ctx context.Context, fn func(ctx context.Context, jobs []dto.Job) (stop bool, err error)) error
}

// Page is one source response and the cursor for the next response.
type Page struct {
	Jobs       []dto.Job
	NextCursor string
}

// PageSource fetches one page for a saved source target.
type PageSource interface {
	FetchPage(ctx context.Context, cursor string) (Page, error)
}

// PageDetailFetcher resolves a page card using its original fetch URL.
type PageDetailFetcher interface {
	GetDetails(ctx context.Context, fetchURL string) (dto.Job, error)
}

type PageFetcher interface {
	FetchPage(context.Context, string) ([]dto.Job, string, error)
}

type DetailFetcher interface {
	GetDetails(ctx context.Context, url string) (dto.Job, error)
}

type PaginatedBase struct {
	cfg     Config
	client  *http.Client
	initErr error
}

func NewBase(cfg Config) PaginatedBase {
	transport, err := proxy.Fetcher(cfg.UseProxy)
	return PaginatedBase{
		cfg:     cfg,
		initErr: err,
		client: &http.Client{Timeout: DefaultTimeout, Transport: transport, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return proxy.ValidateURL(req.Context(), req.URL)
		}},
	}
}

func (b *PaginatedBase) Cfg() Config { return b.cfg }

func (b *PaginatedBase) Client() *http.Client { return b.client }

func (b *PaginatedBase) Get(ctx context.Context, url string) ([]byte, error) {
	return b.do(ctx, http.MethodGet, url, nil)
}

// PostEmptyJSON exists because some ATS list APIs (e.g. Workable) only respond to POST.
func (b *PaginatedBase) PostEmptyJSON(ctx context.Context, url string) ([]byte, error) {
	return b.do(ctx, http.MethodPost, url, []byte("{}"))
}

func (b *PaginatedBase) do(ctx context.Context, method, url string, body []byte) ([]byte, error) {
	if b.initErr != nil {
		return nil, b.initErr
	}
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-GB,en;q=0.9")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return respBody, nil
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
	log := slog.With(slog.String("source", b.cfg.Name))
	log.Info("iterating source", slog.Int("total", total), slog.Int("pages", pages))

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

		log.Debug("fetching page", slog.Int("page", p), slog.Int("of", pages))

		pageURLs, _, err := fetchPage(ctx, p)
		if err != nil {
			log.Error("page failed", slog.Int("page", p), slog.Any("err", err))
			continue
		}

		stop, err = fn(ctx, pageURLs)
		if err != nil {
			return err
		}
		if stop {
			log.Info("early stop", slog.Int("page", p))
			break
		}
	}

	return nil
}

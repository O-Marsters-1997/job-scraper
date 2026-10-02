package sources

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/worker/proxy"
)

const (
	DefaultTimeout = 15 * time.Second
	userAgent      = "Mozilla/5.0 (compatible; job-scraper/1.0)"
)

type Route = proxy.Route

const (
	RouteDirect   = proxy.Direct
	RouteUnlocker = proxy.Unlocker
	RouteTiered   = proxy.Tiered
)

type Config struct {
	Name  string
	Route Route
}

// SnapshotSource is implemented by any source that has snapshot-testable parsers.
// ParseURLs handles list pages; ParseJobDetail handles individual job pages.
// Both return []dto.Job so all snapshot fixtures share a single JSON schema.
type SnapshotSource interface {
	ParseURLs(r io.Reader) ([]dto.Job, error)
	ParseJobDetail(r io.Reader, url string) (dto.Job, error)
}

// Source fetches one page of jobs for a saved source target. cursor is empty
// for the first page; a returned next of "" means the last page. Single-page
// sources (every BoardSource, indeed) always return "".
type Source interface {
	Cfg() Config
	FetchPage(ctx context.Context, cursor string) (jobs []dto.Job, next string, err error)
}

type DetailFetcher interface {
	GetDetails(ctx context.Context, url string) (dto.Job, error)
}

// StatusError is returned by Get and PostEmptyJSON for any non-200 response other than 404 or 410.
type StatusError struct {
	Code       int
	Status     string
	RetryAfter time.Duration
}

func (e *StatusError) Error() string { return "unexpected status " + e.Status }

// BoardResult is what one poll of a Board yields beyond its jobs: an optional next-poll
// hint and an optional company profile for the source's company_profiles row.
type BoardResult struct {
	Jobs       []dto.Job
	NextPollIn time.Duration
	Profile    *dto.CompanyProfile
}

// BoardPoller is implemented by a Source that reports a next-poll hint or a profile.
type BoardPoller interface {
	PollBoard(ctx context.Context) (BoardResult, error)
}

// ErrGone is returned by Get and PostEmptyJSON for a 404 or 410 response.
var ErrGone = errors.New("gone")

// ErrSourceKeyRejected is returned by a keyed Source when the provider refuses
// its API key, so every Target of that Source will fail until the key changes.
var ErrSourceKeyRejected = errors.New("source key rejected")

type PaginatedBase struct {
	cfg     Config
	client  *http.Client
	initErr error
}

func NewBase(cfg Config) PaginatedBase {
	transport, err := proxy.Fetcher(cfg.Route, cfg.Name)
	return PaginatedBase{
		cfg:     cfg,
		initErr: err,
		client: &http.Client{Timeout: DefaultTimeout, Transport: logger.OutboundSpans(transport), CheckRedirect: func(req *http.Request, via []*http.Request) error {
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
	return b.do(ctx, http.MethodGet, url, nil, nil)
}

// PostJSON sends body as JSON; header entries override the defaults.
func (b *PaginatedBase) PostJSON(ctx context.Context, url string, body []byte, header http.Header) ([]byte, error) {
	return b.do(ctx, http.MethodPost, url, body, header)
}

// PostEmptyJSON exists because some ATS list APIs (e.g. Workable) only respond to POST.
func (b *PaginatedBase) PostEmptyJSON(ctx context.Context, url string) ([]byte, error) {
	return b.do(ctx, http.MethodPost, url, []byte("{}"), nil)
}

func (b *PaginatedBase) do(ctx context.Context, method, url string, body []byte, header http.Header) ([]byte, error) {
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
	for k, v := range header {
		req.Header[k] = v
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return nil, fmt.Errorf("%w: status %s", ErrGone, resp.Status)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &StatusError{Code: resp.StatusCode, Status: resp.Status, RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After"))}
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return respBody, nil
}

// parseRetryAfter reads delta-seconds or an HTTP-date, returning zero when absent, unparseable or past.
func parseRetryAfter(v string) time.Duration {
	if secs, err := strconv.Atoi(v); err == nil {
		return max(time.Duration(secs)*time.Second, 0)
	}
	if t, err := http.ParseTime(v); err == nil {
		return max(time.Until(t), 0)
	}
	return 0
}

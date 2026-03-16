package greenhouse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ollymarsters/job-scraper/internal/sources"
)

const (
	baseURL        = "https://boards-api.greenhouse.io/v1/boards"
	defaultTimeout = 10 * time.Second
)

// Config holds configuration for the Greenhouse scraper.
type Config struct {
	// BoardTokens is the list of Greenhouse job board tokens to scrape.
	// Each token corresponds to a company's job board, e.g. "stripe", "airbnb".
	BoardTokens []string
	// HTTPTimeout overrides the default per-request timeout (default: 10s).
	HTTPTimeout time.Duration
}

// Scraper implements sources.Source for the Greenhouse Job Board API.
type Scraper struct {
	cfg    Config
	client *http.Client
}

// New constructs a Greenhouse Scraper. Returns an error if no board tokens are provided.
func New(cfg Config) (*Scraper, error) {
	if len(cfg.BoardTokens) == 0 {
		return nil, errors.New("greenhouse: at least one board token is required")
	}
	timeout := cfg.HTTPTimeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	return &Scraper{
		cfg:    cfg,
		client: &http.Client{Timeout: timeout},
	}, nil
}

// Name implements sources.Source.
func (s *Scraper) Name() string { return "greenhouse" }

// FetchURLs implements sources.Source. It calls the Greenhouse Job Board API
// once per configured board token and returns all open job URLs.
// Per-token errors are collected; partial results are returned alongside errors.
func (s *Scraper) FetchURLs(ctx context.Context) ([]string, error) {
	var (
		all  []string
		errs []error
	)

	for _, token := range s.cfg.BoardTokens {
		if err := ctx.Err(); err != nil {
			return all, err
		}

		slog.Debug("fetching board", slog.String("source", s.Name()), slog.String("token", token))

		urls, err := s.fetchBoard(ctx, token)
		if err != nil {
			slog.Error("board fetch failed", slog.String("source", s.Name()), slog.String("token", token), slog.Any("err", err))
			errs = append(errs, fmt.Errorf("greenhouse/%s: %w", token, err))
			continue
		}

		slog.Info("board fetch complete", slog.String("source", s.Name()), slog.String("token", token), slog.Int("count", len(urls)))
		all = append(all, urls...)
	}

	return all, errors.Join(errs...)
}

// Iterate implements sources.Source. Greenhouse boards are not paginated at
// the scrape level so this fetches all URLs then passes them through the filter.
func (s *Scraper) Iterate(ctx context.Context, filter sources.URLFilter) ([]string, error) {
	urls, err := s.FetchURLs(ctx)
	if err != nil {
		return nil, err
	}
	return filter(ctx, urls)
}

func (s *Scraper) fetchBoard(ctx context.Context, token string) ([]string, error) {
	url := fmt.Sprintf("%s/%s/jobs", baseURL, token)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}

	var payload struct {
		Jobs []struct {
			AbsoluteURL string `json:"absolute_url"`
		} `json:"jobs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	urls := make([]string, 0, len(payload.Jobs))
	for _, gj := range payload.Jobs {
		if gj.AbsoluteURL != "" {
			urls = append(urls, gj.AbsoluteURL)
		}
	}

	return urls, nil
}

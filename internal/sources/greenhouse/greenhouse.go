package greenhouse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// FetchJobs implements sources.Source. It calls the Greenhouse Job Board API
// once per configured board token and returns all open job postings.
// Per-token errors are collected; partial results are returned alongside errors.
func (s *Scraper) FetchJobs(ctx context.Context) ([]sources.Job, error) {
	var (
		all  []sources.Job
		errs []error
	)

	for _, token := range s.cfg.BoardTokens {
		if err := ctx.Err(); err != nil {
			return all, err
		}

		jobs, err := s.fetchBoard(ctx, token)
		if err != nil {
			errs = append(errs, fmt.Errorf("greenhouse/%s: %w", token, err))
			continue
		}

		all = append(all, jobs...)
	}

	return all, errors.Join(errs...)
}

func (s *Scraper) fetchBoard(ctx context.Context, token string) ([]sources.Job, error) {
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
			ID        int64  `json:"id"`
			Title     string `json:"title"`
			UpdatedAt string `json:"updated_at"`
			Location  struct {
				Name string `json:"name"`
			} `json:"location"`
			AbsoluteURL string `json:"absolute_url"`
		} `json:"jobs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	jobs := make([]sources.Job, 0, len(payload.Jobs))
	for _, gj := range payload.Jobs {
		updatedAt, _ := time.Parse(time.RFC3339, gj.UpdatedAt)
		jobs = append(jobs, sources.Job{
			Title:       gj.Title,
			Location:    gj.Location.Name,
			URL:         gj.AbsoluteURL,
			CompanySlug: token,
			Source:      s.Name(),
			UpdatedAt:   updatedAt,
		})
	}

	return jobs, nil
}

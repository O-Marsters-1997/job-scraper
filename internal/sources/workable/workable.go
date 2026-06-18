package workable

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

// Config holds Workable-specific settings.
type Config struct {
	// Boards is the list of Workable company slugs to scrape (e.g. "acmecorp").
	Boards []string
}

type Scraper struct {
	sources.PaginatedBase
	cfg Config
}

var _ sources.Source = (*Scraper)(nil)

func New(cfg Config) *Scraper {
	return &Scraper{
		PaginatedBase: sources.NewBase(sources.Config{
			Name:              "workable",
			URLPrefix:         "https://apply.workable.com",
			Schedule:          "0 */6 * * *",
			MinScrapeInterval: 5 * time.Hour,
		}),
		cfg: cfg,
	}
}

func (s *Scraper) Iterate(ctx context.Context, fn func(context.Context, []dto.Job) (bool, error)) error {
	for _, token := range s.cfg.Boards {
		jobs, err := s.fetchBoard(ctx, token)
		if err != nil {
			return fmt.Errorf("workable: board %s: %w", token, err)
		}
		stop, err := fn(ctx, jobs)
		if err != nil {
			return err
		}
		if stop {
			return nil
		}
	}
	return nil
}

type boardResponse struct {
	Results []jobResult `json:"results"`
}

type jobResult struct {
	Shortcode       string      `json:"shortcode"`
	Title           string      `json:"title"`
	Location        jobLocation `json:"location"`
	URL             string      `json:"url"`
	FullDescription string      `json:"full_description"`
}

type jobLocation struct {
	City string `json:"city"`
}

func (s *Scraper) fetchBoard(ctx context.Context, token string) ([]dto.Job, error) {
	url := fmt.Sprintf("https://apply.workable.com/api/v3/accounts/%s/jobs", token)
	body, err := s.Get(ctx, url)
	if err != nil {
		return nil, err
	}

	var resp boardResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	jobs := make([]dto.Job, 0, len(resp.Results))
	for _, r := range resp.Results {
		jobs = append(jobs, dto.Job{
			Title:       r.Title,
			Location:    r.Location.City,
			URL:         r.URL,
			CompanySlug: token,
			Source:      "workable",
			Description: r.FullDescription,
			SalaryRaw:   "",
			UpdatedAt:   time.Now().UTC(),
		})
	}
	return jobs, nil
}

package recruitee

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

// Config holds Recruitee-specific settings.
type Config struct {
	// Boards is the list of Recruitee company subdomains to scrape (e.g. "acmecorp").
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
			Name:              "recruitee",
			URLPrefix:         "https://recruitee.com",
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
			return fmt.Errorf("recruitee: board %s: %w", token, err)
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
	Offers []offer `json:"offers"`
}

type offer struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Location    string `json:"location"`
	CareersURL  string `json:"careers_url"`
	Description string `json:"description"`
}

func (s *Scraper) fetchBoard(ctx context.Context, token string) ([]dto.Job, error) {
	url := fmt.Sprintf("https://%s.recruitee.com/api/offers/", token)
	body, err := s.Get(ctx, url)
	if err != nil {
		return nil, err
	}

	var resp boardResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	jobs := make([]dto.Job, 0, len(resp.Offers))
	for _, o := range resp.Offers {
		jobs = append(jobs, dto.Job{
			Title:       o.Title,
			Location:    o.Location,
			URL:         o.CareersURL,
			CompanySlug: token,
			Source:      "recruitee",
			Description: o.Description,
			SalaryRaw:   "",
			UpdatedAt:   time.Now().UTC(),
		})
	}
	return jobs, nil
}

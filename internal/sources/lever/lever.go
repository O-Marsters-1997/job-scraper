package lever

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

// Config holds Lever-specific settings.
type Config struct {
	// Boards is the list of Lever company slugs to scrape (e.g. "acmecorp").
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
			Name:              "lever",
			URLPrefix:         "https://jobs.lever.co",
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
			return fmt.Errorf("lever: board %s: %w", token, err)
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

type posting struct {
	ID               string            `json:"id"`
	Text             string            `json:"text"`
	Categories       postingCategories `json:"categories"`
	HostedURL        string            `json:"hostedUrl"`
	DescriptionPlain string            `json:"descriptionPlain"`
	CreatedAt        int64             `json:"createdAt"`
}

type postingCategories struct {
	Location string `json:"location"`
}

func (s *Scraper) fetchBoard(ctx context.Context, token string) ([]dto.Job, error) {
	url := fmt.Sprintf("https://api.lever.co/v0/postings/%s?mode=json", token)
	body, err := s.Get(ctx, url)
	if err != nil {
		return nil, err
	}

	var postings []posting
	if err := json.Unmarshal(body, &postings); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	jobs := make([]dto.Job, 0, len(postings))
	for _, p := range postings {
		updatedAt := time.Now().UTC()
		if p.CreatedAt != 0 {
			updatedAt = time.UnixMilli(p.CreatedAt).UTC()
		}

		jobs = append(jobs, dto.Job{
			Title:       p.Text,
			Location:    p.Categories.Location,
			URL:         p.HostedURL,
			CompanySlug: token,
			Source:      "lever",
			Description: p.DescriptionPlain,
			SalaryRaw:   "",
			UpdatedAt:   updatedAt,
		})
	}
	return jobs, nil
}

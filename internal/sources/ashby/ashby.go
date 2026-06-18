package ashby

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

// Config holds Ashby-specific settings.
type Config struct {
	// Boards is the list of Ashby company slugs to scrape (e.g. "acmecorp").
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
			Name:              "ashby",
			URLPrefix:         "https://jobs.ashbyhq.com",
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
			return fmt.Errorf("ashby: board %s: %w", token, err)
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
	JobPostings []jobPosting `json:"jobPostings"`
}

type jobPosting struct {
	ID              string      `json:"id"`
	Title           string      `json:"title"`
	Location        jobLocation `json:"location"`
	JobURL          string      `json:"jobUrl"`
	DescriptionHTML string      `json:"descriptionHtml"`
	PublishedDate   string      `json:"publishedDate"`
}

type jobLocation struct {
	Name string `json:"name"`
}

func (s *Scraper) fetchBoard(ctx context.Context, token string) ([]dto.Job, error) {
	url := fmt.Sprintf("https://api.ashbyhq.com/posting-api/job-board/%s", token)
	body, err := s.Get(ctx, url)
	if err != nil {
		return nil, err
	}

	var resp boardResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	jobs := make([]dto.Job, 0, len(resp.JobPostings))
	for _, jp := range resp.JobPostings {
		updatedAt := time.Now().UTC()
		if jp.PublishedDate != "" {
			if t, err := time.Parse(time.RFC3339, jp.PublishedDate); err == nil {
				updatedAt = t
			}
		}

		jobs = append(jobs, dto.Job{
			Title:       jp.Title,
			Location:    jp.Location.Name,
			URL:         jp.JobURL,
			CompanySlug: token,
			Source:      "ashby",
			Description: jp.DescriptionHTML,
			SalaryRaw:   "",
			UpdatedAt:   updatedAt,
		})
	}
	return jobs, nil
}

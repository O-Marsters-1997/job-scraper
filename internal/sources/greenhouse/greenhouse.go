package greenhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

// Config holds Greenhouse-specific settings.
type Config struct {
	// Boards is the list of Greenhouse board tokens to scrape (e.g. "acmecorp").
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
			Name:              "greenhouse",
			URLPrefix:         "https://boards.greenhouse.io",
			Schedule:          "0 */6 * * *",
			MinScrapeInterval: 5 * time.Hour,
		}),
		cfg: cfg,
	}
}

func (s *Scraper) NeedsDetail() bool { return false }

// GetDetails is a no-op for ATS sources — jobs arrive fully populated from Iterate.
func (s *Scraper) GetDetails(_ context.Context, _ string) (dto.Job, error) {
	return dto.Job{}, fmt.Errorf("greenhouse: GetDetails must not be called (NeedsDetail=false)")
}

func (s *Scraper) Iterate(ctx context.Context, fn func(context.Context, []dto.Job) (bool, error)) error {
	for _, token := range s.cfg.Boards {
		jobs, err := s.fetchBoard(ctx, token)
		if err != nil {
			return fmt.Errorf("greenhouse: board %s: %w", token, err)
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

// boardResponse mirrors the Greenhouse boards API v1 response.
type boardResponse struct {
	Jobs []boardJob `json:"jobs"`
}

type boardJob struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Location    jobLocation `json:"location"`
	AbsoluteURL string     `json:"absolute_url"`
	Content     string     `json:"content"`
	UpdatedAt   string     `json:"updated_at"`
}

type jobLocation struct {
	Name string `json:"name"`
}

func (s *Scraper) fetchBoard(ctx context.Context, token string) ([]dto.Job, error) {
	url := fmt.Sprintf("https://boards-api.greenhouse.io/v1/boards/%s/jobs?content=true", token)
	body, err := s.Get(ctx, url)
	if err != nil {
		return nil, err
	}

	var resp boardResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	jobs := make([]dto.Job, 0, len(resp.Jobs))
	for _, bj := range resp.Jobs {
		updatedAt := time.Now().UTC()
		if bj.UpdatedAt != "" {
			if t, err := time.Parse(time.RFC3339, bj.UpdatedAt); err == nil {
				updatedAt = t
			}
		}

		jobs = append(jobs, dto.Job{
			Title:       bj.Title,
			Location:    bj.Location.Name,
			URL:         bj.AbsoluteURL,
			CompanySlug: token,
			Source:      "greenhouse",
			Description: bj.Content,
			SalaryRaw:   "",
			UpdatedAt:   updatedAt,
		})
	}
	return jobs, nil
}

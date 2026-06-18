package personio

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

// Config holds Personio-specific settings.
type Config struct {
	// Boards is the list of Personio company subdomains to scrape (e.g. "acmecorp").
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
			Name:              "personio",
			URLPrefix:         "https://personio.de",
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
			return fmt.Errorf("personio: board %s: %w", token, err)
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

type workzagJobs struct {
	XMLName xml.Name      `xml:"workzag-jobs"`
	Jobs    []personioJob `xml:"job"`
}

type personioJob struct {
	JobPosition    string `xml:"jobPosition"`
	Office         string `xml:"office"`
	ApplyOnline    string `xml:"applyOnline"`
	JobDescription string `xml:"jobDescription"`
}

func (s *Scraper) fetchBoard(ctx context.Context, token string) ([]dto.Job, error) {
	url := fmt.Sprintf("https://%s.personio.de/xml", token)
	body, err := s.Get(ctx, url)
	if err != nil {
		return nil, err
	}

	var root workzagJobs
	if err := xml.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("parse xml: %w", err)
	}

	jobs := make([]dto.Job, 0, len(root.Jobs))
	for _, pj := range root.Jobs {
		jobs = append(jobs, dto.Job{
			Title:       pj.JobPosition,
			Location:    pj.Office,
			URL:         pj.ApplyOnline,
			CompanySlug: token,
			Source:      "personio",
			Description: pj.JobDescription,
			SalaryRaw:   "",
			UpdatedAt:   time.Now().UTC(),
		})
	}
	return jobs, nil
}

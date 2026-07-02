package workable

import (
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

func New(cfg Config) *sources.BoardSource {
	return sources.NewBoardSource(cfg.Boards, sources.BoardSpec{
		Name:      "workable",
		URLPrefix: "https://apply.workable.com",
		URL: func(token string) string {
			return fmt.Sprintf("https://apply.workable.com/api/v3/accounts/%s/jobs", token)
		},
		Parse: parse,
	})
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

func parse(body []byte, token string) ([]dto.Job, error) {
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
			UpdatedAt:   time.Now().UTC(),
		})
	}
	return jobs, nil
}

package ashby

import (
	"encoding/json"
	"fmt"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

// Config holds Ashby-specific settings.
type Config struct {
	// Boards is the list of Ashby company slugs to scrape (e.g. "acmecorp").
	Boards []string
}

func New(cfg Config) *sources.BoardSource {
	return sources.NewBoardSource(cfg.Boards, sources.BoardSpec{
		Name:      "ashby",
		URLPrefix: "https://jobs.ashbyhq.com",
		URL: func(token string) string {
			return fmt.Sprintf("https://api.ashbyhq.com/posting-api/job-board/%s", token)
		},
		Parse: parse,
	})
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

func parse(body []byte, token string) ([]dto.Job, error) {
	var resp boardResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	jobs := make([]dto.Job, 0, len(resp.JobPostings))
	for _, jp := range resp.JobPostings {
		jobs = append(jobs, dto.Job{
			Title:       jp.Title,
			Location:    jp.Location.Name,
			URL:         jp.JobURL,
			CompanySlug: token,
			Source:      "ashby",
			Description: jp.DescriptionHTML,
			UpdatedAt:   sources.RFC3339OrNow(jp.PublishedDate),
		})
	}
	return jobs, nil
}

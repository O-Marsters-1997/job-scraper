package ashby

import (
	"encoding/json"
	"fmt"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

type Config struct {
	// Boards is the list of Ashby company slugs to scrape (e.g. "acmecorp").
	Boards []string
}

func New(cfg Config) *sources.BoardSource {
	return sources.NewBoardSource(cfg.Boards, sources.BoardSpec{
		Name: "ashby",
		URL: func(token string) string {
			return fmt.Sprintf("https://api.ashbyhq.com/posting-api/job-board/%s", token)
		},
		Parse: parse,
	})
}

type boardResponse struct {
	Jobs []jobPosting `json:"jobs"`
}

type jobPosting struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Location        string `json:"location"`
	JobURL          string `json:"jobUrl"`
	DescriptionHTML string `json:"descriptionHtml"`
	PublishedAt     string `json:"publishedAt"`
}

func parse(body []byte, token string) ([]dto.Job, error) {
	var resp boardResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	jobs := make([]dto.Job, 0, len(resp.Jobs))
	for _, jp := range resp.Jobs {
		jobs = append(jobs, dto.Job{
			Title:             jp.Title,
			Location:          jp.Location,
			URL:               jp.JobURL,
			CompanySlug:       token,
			ProviderPostingID: jp.ID,
			Source:            "ashby",
			Description:       jp.DescriptionHTML,
			UpdatedAt:         sources.RFC3339OrNow(jp.PublishedAt),
		})
	}
	return jobs, nil
}

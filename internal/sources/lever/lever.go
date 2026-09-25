package lever

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

type Config struct {
	// Boards is the list of Lever company slugs to scrape (e.g. "acmecorp").
	Boards []string
}

func New(cfg Config) *sources.BoardSource {
	return sources.NewBoardSource(cfg.Boards, sources.BoardSpec{
		Name: "lever",
		URL: func(token string) string {
			return fmt.Sprintf("https://api.lever.co/v0/postings/%s?mode=json", token)
		},
		Parse: parse,
	})
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

func parse(body []byte, token string) ([]dto.Job, error) {
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
			Title:             p.Text,
			Location:          p.Categories.Location,
			URL:               p.HostedURL,
			CompanySlug:       token,
			ProviderPostingID: p.ID,
			Source:            "lever",
			Description:       p.DescriptionPlain,
			UpdatedAt:         updatedAt,
		})
	}
	return jobs, nil
}

package ashby

import (
	"encoding/json"
	"fmt"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

// New builds an Ashby source for one company slug (e.g. "acmecorp").
func New(token string) *sources.BoardSource {
	return sources.NewBoardSource(sources.BoardSpec{
		Name:        "ashby",
		URL:         fmt.Sprintf("https://api.ashbyhq.com/posting-api/job-board/%s", token),
		CompanySlug: token,
		Parse:       parse,
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

func parse(body []byte) ([]dto.Job, error) {
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
			ProviderPostingID: jp.ID,
			Description:       jp.DescriptionHTML,
			UpdatedAt:         sources.RFC3339OrNow(jp.PublishedAt),
		})
	}
	return jobs, nil
}

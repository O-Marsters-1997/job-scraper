package greenhouse

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

// New builds a Greenhouse source for one board token (e.g. "acmecorp").
func New(token string) *sources.BoardSource {
	return sources.NewBoardSource(sources.BoardSpec{
		Name:        "greenhouse",
		URL:         fmt.Sprintf("https://boards-api.greenhouse.io/v1/boards/%s/jobs?content=true", token),
		CompanySlug: token,
		Parse:       parse,
		Count:       count,
	})
}

// boardResponse mirrors the Greenhouse boards API v1 response.
type boardResponse struct {
	Jobs []boardJob `json:"jobs"`
}

type boardJob struct {
	ID          int64       `json:"id"`
	Title       string      `json:"title"`
	Location    jobLocation `json:"location"`
	AbsoluteURL string      `json:"absolute_url"`
	Content     string      `json:"content"`
	UpdatedAt   string      `json:"updated_at"`
}

type jobLocation struct {
	Name string `json:"name"`
}

func count(body []byte) (int, error) {
	var resp struct {
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("parse json: %w", err)
	}
	return resp.Meta.Total, nil
}

func parse(body []byte) ([]dto.Job, error) {
	var resp boardResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	jobs := make([]dto.Job, 0, len(resp.Jobs))
	for _, bj := range resp.Jobs {
		jobs = append(jobs, dto.Job{
			Title:             bj.Title,
			Location:          bj.Location.Name,
			URL:               bj.AbsoluteURL,
			ProviderPostingID: strconv.FormatInt(bj.ID, 10),
			Description:       bj.Content,
			UpdatedAt:         sources.RFC3339OrNow(bj.UpdatedAt),
		})
	}
	return jobs, nil
}

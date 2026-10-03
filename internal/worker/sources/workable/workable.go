package workable

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

// New builds a Workable source for one company slug (e.g. "acmecorp").
func New(token string) *sources.BoardSource {
	return sources.NewBoardSource(sources.BoardSpec{
		Name:        "workable",
		Post:        true, // Workable's list API 404s on GET; only POST responds.
		URL:         fmt.Sprintf("https://apply.workable.com/api/v3/accounts/%s/jobs", token),
		CompanySlug: token,
		Parse:       func(body []byte) ([]dto.Job, error) { return parse(body, token) },
		Count:       count,
	})
}

type boardResponse struct {
	Results []jobResult `json:"results"`
}

type jobResult struct {
	Shortcode string      `json:"shortcode"`
	Title     string      `json:"title"`
	Location  jobLocation `json:"location"`
}

type jobLocation struct {
	City string `json:"city"`
}

func count(body []byte) (int, error) {
	var resp struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("parse json: %w", err)
	}
	return resp.Total, nil
}

// Workable's list API returns no job URL or description; the URL here is
// built from the shortcode.
func parse(body []byte, token string) ([]dto.Job, error) {
	var resp boardResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	jobs := make([]dto.Job, 0, len(resp.Results))
	for _, r := range resp.Results {
		jobs = append(jobs, dto.Job{
			Title:             r.Title,
			Location:          r.Location.City,
			URL:               fmt.Sprintf("https://apply.workable.com/%s/j/%s/", token, r.Shortcode),
			ProviderPostingID: r.Shortcode,
			UpdatedAt:         time.Now().UTC(),
		})
	}
	return jobs, nil
}

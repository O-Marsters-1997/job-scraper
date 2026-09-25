package workable

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

type Config struct {
	// Boards is the list of Workable company slugs to scrape (e.g. "acmecorp").
	Boards []string
}

func New(cfg Config) *sources.BoardSource {
	return sources.NewBoardSource(cfg.Boards, sources.BoardSpec{
		Name: "workable",
		Post: true, // Workable's list API 404s on GET; only POST responds.
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
	Shortcode string      `json:"shortcode"`
	Title     string      `json:"title"`
	Location  jobLocation `json:"location"`
}

type jobLocation struct {
	City string `json:"city"`
}

// The list carries no job URL or description, so the URL is built from the
// shortcode and the description is left empty (the heuristic relevance gate
// scores on title+location only).
// ponytail: description omitted; add a per-job detail fetch if a scorer needs it.
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
			CompanySlug:       token,
			ProviderPostingID: r.Shortcode,
			Source:            "workable",
			UpdatedAt:         time.Now().UTC(),
		})
	}
	return jobs, nil
}

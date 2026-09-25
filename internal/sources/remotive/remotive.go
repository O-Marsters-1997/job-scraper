package remotive

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

// Config holds the keywords to filter the Remotive feed by, one per enabled
// remotive target's value.
type Config struct {
	Keywords []string
}

func New(cfg Config) *sources.BoardSource {
	return sources.NewBoardSource([]string{""}, sources.BoardSpec{
		Name: "remotive",
		URL:  func(string) string { return "https://remotive.com/api/remote-jobs" },
		Parse: func(body []byte, _ string) ([]dto.Job, error) {
			jobs, err := parse(body)
			return sources.FilterByKeywords(jobs, cfg.Keywords), err
		},
	})
}

// feedResponse mirrors the Remotive public jobs API response.
type feedResponse struct {
	Jobs []feedJob `json:"jobs"`
}

type feedJob struct {
	Title           string `json:"title"`
	URL             string `json:"url"`
	CompanyName     string `json:"company_name"`
	Location        string `json:"candidate_required_location"`
	Description     string `json:"description"`
	Salary          string `json:"salary"`
	PublicationDate string `json:"publication_date"`
}

// ponytail: URL-only dedup; a company posting both on its ATS board and on
// this feed yields two rows for the same role. Add an {ats}:{company}:{job_id}
// key only if measured dup rate is material.
func parse(body []byte) ([]dto.Job, error) {
	var resp feedResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	jobs := make([]dto.Job, 0, len(resp.Jobs))
	for _, fj := range resp.Jobs {
		jobs = append(jobs, dto.Job{
			Title:           fj.Title,
			Location:        fj.Location,
			URL:             fj.URL,
			CompanySlug:     sources.Slugify(fj.CompanyName),
			Source:          "remotive",
			Description:     fj.Description,
			SalaryRaw:       fj.Salary,
			WorkArrangement: "remote",
			UpdatedAt:       parsePublicationDate(fj.PublicationDate),
		})
	}
	return jobs, nil
}

// publicationDateLayout matches Remotive's publication_date, e.g.
// "2026-07-04T16:53:04" — no timezone offset, so it doesn't fit
// sources.RFC3339OrNow. Remotive's own docs treat these timestamps as UTC.
const publicationDateLayout = "2006-01-02T15:04:05"

func parsePublicationDate(raw string) time.Time {
	if raw != "" {
		if t, err := time.Parse(publicationDateLayout, raw); err == nil {
			return t.UTC()
		}
	}
	return time.Now().UTC()
}

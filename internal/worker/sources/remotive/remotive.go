package remotive

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/slug"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

// New builds a Remotive source filtered to one keyword (empty means no filter).
func New(keyword string) *sources.BoardSource {
	return sources.NewBoardSource(sources.BoardSpec{
		Name: "remotive",
		URL:  "https://remotive.com/api/remote-jobs",
		Parse: func(body []byte) ([]dto.Job, error) {
			jobs, err := parse(body)
			return sources.FilterByKeywords(jobs, keywordList(keyword)), err
		},
	})
}

func keywordList(keyword string) []string {
	if keyword == "" {
		return nil
	}
	return []string{keyword}
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
			CompanySlug:     slug.Company(fj.CompanyName),
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

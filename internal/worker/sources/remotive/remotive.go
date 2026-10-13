package remotive

import (
	"encoding/json"
	"fmt"

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
			PostedAt:        sources.TimeOrNil(publicationDateLayout, fj.PublicationDate),
		})
	}
	return jobs, nil
}

const publicationDateLayout = "2006-01-02T15:04:05"

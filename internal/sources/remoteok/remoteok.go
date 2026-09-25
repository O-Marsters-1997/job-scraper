package remoteok

import (
	"encoding/json"
	"fmt"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/slug"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

// Config holds the keywords to filter the RemoteOK feed by, one per enabled
// remoteok target's value.
type Config struct {
	Keywords []string
}

func New(cfg Config) *sources.BoardSource {
	return sources.NewBoardSource([]string{""}, sources.BoardSpec{
		Name: "remoteok",
		URL:  func(string) string { return "https://remoteok.com/api" },
		Parse: func(body []byte, _ string) ([]dto.Job, error) {
			jobs, err := parse(body)
			return sources.FilterByKeywords(jobs, cfg.Keywords), err
		},
	})
}

// feedJob mirrors one entry of the RemoteOK API response. The feed's first
// element is a legal notice with no id/slug/url, not a job.
type feedJob struct {
	ID          string `json:"id"`
	Position    string `json:"position"`
	Company     string `json:"company"`
	Location    string `json:"location"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Date        string `json:"date"`
	SalaryMin   int    `json:"salary_min"`
	SalaryMax   int    `json:"salary_max"`
}

func parse(body []byte) ([]dto.Job, error) {
	var raw []feedJob
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	jobs := make([]dto.Job, 0, len(raw))
	for _, fj := range raw {
		if fj.ID == "" {
			continue
		}
		jobs = append(jobs, dto.Job{
			Title:           fj.Position,
			Location:        fj.Location,
			URL:             fj.URL,
			CompanySlug:     slug.Make(fj.Company),
			Source:          "remoteok",
			Description:     fj.Description,
			SalaryRaw:       formatSalary(fj.SalaryMin, fj.SalaryMax),
			WorkArrangement: "remote",
			UpdatedAt:       sources.RFC3339OrNow(fj.Date),
		})
	}
	return jobs, nil
}

// formatSalary renders RemoteOK's numeric salary_min/salary_max as a raw
// display string. Both fields are 0 when the poster didn't specify a salary.
func formatSalary(min, max int) string {
	if min == 0 && max == 0 {
		return ""
	}
	return fmt.Sprintf("$%d - $%d", min, max)
}

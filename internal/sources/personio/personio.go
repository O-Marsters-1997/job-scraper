package personio

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

// Config holds Personio-specific settings.
type Config struct {
	// Boards is the list of Personio company subdomains to scrape (e.g. "acmecorp").
	Boards []string
}

func New(cfg Config) *sources.BoardSource {
	return sources.NewBoardSource(cfg.Boards, sources.BoardSpec{
		Name:      "personio",
		URLPrefix: "https://personio.de",
		URL: func(token string) string {
			return fmt.Sprintf("https://%s.personio.de/xml", token)
		},
		Parse: parse,
	})
}

type workzagJobs struct {
	XMLName xml.Name      `xml:"workzag-jobs"`
	Jobs    []personioJob `xml:"job"`
}

type personioJob struct {
	JobPosition    string `xml:"jobPosition"`
	Office         string `xml:"office"`
	ApplyOnline    string `xml:"applyOnline"`
	JobDescription string `xml:"jobDescription"`
}

func parse(body []byte, token string) ([]dto.Job, error) {
	var root workzagJobs
	if err := xml.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("parse xml: %w", err)
	}

	jobs := make([]dto.Job, 0, len(root.Jobs))
	for _, pj := range root.Jobs {
		jobs = append(jobs, dto.Job{
			Title:       pj.JobPosition,
			Location:    pj.Office,
			URL:         pj.ApplyOnline,
			CompanySlug: token,
			Source:      "personio",
			Description: pj.JobDescription,
			UpdatedAt:   time.Now().UTC(),
		})
	}
	return jobs, nil
}

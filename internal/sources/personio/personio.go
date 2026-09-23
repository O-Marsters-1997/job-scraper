package personio

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

type Config struct {
	// Boards is the list of Personio company subdomains to scrape (e.g. "acmecorp").
	Boards []string
}

func New(cfg Config) *sources.BoardSource {
	return sources.NewBoardSource(cfg.Boards, sources.BoardSpec{
		Name:      "personio",
		URLPrefix: "https://",
		URL: func(token string) string {
			return fmt.Sprintf("https://%s.jobs.personio.com/xml", token)
		},
		Parse: parse,
	})
}

type workzagJobs struct {
	XMLName   xml.Name      `xml:"workzag-jobs"`
	Positions []personioJob `xml:"position"`
}

type personioJob struct {
	ID              string           `xml:"id"`
	Name            string           `xml:"name"`
	Office          string           `xml:"office"`
	JobDescriptions []jobDescription `xml:"jobDescriptions>jobDescription"`
}

type jobDescription struct {
	Value string `xml:"value"`
}

func parse(body []byte, token string) ([]dto.Job, error) {
	var root workzagJobs
	if err := xml.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("parse xml: %w", err)
	}

	jobs := make([]dto.Job, 0, len(root.Positions))
	for _, p := range root.Positions {
		var sb strings.Builder
		for _, d := range p.JobDescriptions {
			sb.WriteString(d.Value)
		}
		jobs = append(jobs, dto.Job{
			Title:             p.Name,
			Location:          p.Office,
			URL:               fmt.Sprintf("https://%s.jobs.personio.com/job/%s", token, p.ID),
			CompanySlug:       token,
			ProviderPostingID: p.ID,
			Source:            "personio",
			Description:       sb.String(),
			UpdatedAt:         time.Now().UTC(),
		})
	}
	return jobs, nil
}

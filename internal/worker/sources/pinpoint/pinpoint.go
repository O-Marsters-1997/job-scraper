package pinpoint

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

// New builds a Pinpoint source for one company subdomain (e.g. "acmecorp").
func New(token string) *sources.BoardSource {
	return sources.NewBoardSource(sources.BoardSpec{
		Name:        "pinpoint",
		URL:         fmt.Sprintf("https://%s.pinpointhq.com/postings.json", token),
		CompanySlug: token,
		Header:      http.Header{"Accept": {"application/json"}},
		Parse:       parse,
		Count:       sources.JSONArrayLen("data"),
	})
}

type boardResponse struct {
	Data []posting `json:"data"`
}

type posting struct {
	ID                    string  `json:"id"`
	Title                 string  `json:"title"`
	URL                   string  `json:"url"`
	Description           string  `json:"description"`
	KeyResponsibilities   string  `json:"key_responsibilities"`
	SkillsKnowledge       string  `json:"skills_knowledge_expertise"`
	WorkplaceType         string  `json:"workplace_type"`
	CompensationVisible   bool    `json:"compensation_visible"`
	CompensationMinimum   float64 `json:"compensation_minimum"`
	CompensationMaximum   float64 `json:"compensation_maximum"`
	CompensationCurrency  string  `json:"compensation_currency"`
	CompensationFrequency string  `json:"compensation_frequency"`
	Location              struct {
		City string `json:"city"`
		Name string `json:"name"`
	} `json:"location"`
}

func (p posting) salaryRaw() string {
	if !p.CompensationVisible {
		return ""
	}
	return sources.SalaryRange(int(p.CompensationMinimum), int(p.CompensationMaximum), p.CompensationCurrency, p.CompensationFrequency)
}

func (p posting) workArrangement() string {
	switch p.WorkplaceType {
	case "remote", "hybrid", "onsite":
		return p.WorkplaceType
	default:
		return ""
	}
}

func (p posting) location() string {
	if p.Location.City == "" || p.Location.City == p.Location.Name {
		return p.Location.Name
	}
	if p.Location.Name == "" {
		return p.Location.City
	}
	return p.Location.City + ", " + p.Location.Name
}

func (p posting) fullDescription() string {
	var parts []string
	for _, section := range []string{p.Description, p.KeyResponsibilities, p.SkillsKnowledge} {
		if section != "" {
			parts = append(parts, section)
		}
	}
	return strings.Join(parts, "\n")
}

func parse(body []byte) ([]dto.Job, error) {
	var resp boardResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	jobs := make([]dto.Job, 0, len(resp.Data))
	for _, p := range resp.Data {
		jobs = append(jobs, dto.Job{
			Title:             p.Title,
			Location:          p.location(),
			URL:               p.URL,
			ProviderPostingID: p.ID,
			Description:       p.fullDescription(),
			SalaryRaw:         p.salaryRaw(),
			WorkArrangement:   p.workArrangement(),
			UpdatedAt:         time.Now().UTC(),
		})
	}
	return jobs, nil
}

package hibob

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

// New builds a HiBob source for one company subdomain (e.g. "unidays"). HiBob rejects
// the request with 401 unless the subdomain is also sent in the companyidentifier header.
func New(token string) *sources.BoardSource {
	header := http.Header{
		"Accept":            {"application/json"},
		"Companyidentifier": {token},
	}
	return sources.NewBoardSource(sources.BoardSpec{
		Name:        "hibob",
		URL:         fmt.Sprintf("https://%s.careers.hibob.com/api/job-ad", token),
		CompanySlug: token,
		Header:      header,
		Parse:       func(body []byte) ([]dto.Job, error) { return parse(token, body) },
		Count:       sources.JSONArrayLen("jobAdDetails"),
	})
}

type boardResponse struct {
	JobAdDetails []jobAd `json:"jobAdDetails"`
}

type jobAd struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Site             string   `json:"site"`
	Country          string   `json:"country"`
	Description      string   `json:"description"`
	Requirements     string   `json:"requirements"`
	Responsibilities string   `json:"responsibilities"`
	Benefits         string   `json:"benefits"`
	PublishedAt      string   `json:"publishedAt"`
	WorkspaceTypeID  string   `json:"workspaceTypeId"`
	MinSalary        *float64 `json:"payTransparencyMinSalary"`
	MaxSalary        *float64 `json:"payTransparencyMaxSalary"`
	SalaryCurrency   string   `json:"payTransparencySalaryCurrency"`
	SalaryPayPeriod  string   `json:"payTransparencySalaryPayPeriod"`
}

var salaryPeriods = map[string]string{
	"Annual":  "year",
	"Monthly": "month",
	"Weekly":  "week",
	"Daily":   "day",
	"Hourly":  "hour",
}

func (a jobAd) salaryRaw() string {
	var low, high int
	if a.MinSalary != nil {
		low = int(*a.MinSalary)
	}
	if a.MaxSalary != nil {
		high = int(*a.MaxSalary)
	}
	return sources.SalaryRange(low, high, a.SalaryCurrency, salaryPeriods[a.SalaryPayPeriod])
}

func (a jobAd) workArrangement() string {
	switch a.WorkspaceTypeID {
	case "remote", "hybrid":
		return a.WorkspaceTypeID
	case "on_site":
		return "onsite"
	default:
		return ""
	}
}

func (a jobAd) location() string {
	if a.Site == "" || a.Site == a.Country {
		return a.Country
	}
	if a.Country == "" {
		return a.Site
	}
	return a.Site + ", " + a.Country
}

func (a jobAd) fullDescription() string {
	var parts []string
	for _, section := range []string{a.Description, a.Responsibilities, a.Requirements, a.Benefits} {
		if section != "" {
			parts = append(parts, section)
		}
	}
	return strings.Join(parts, "\n")
}

func parse(token string, body []byte) ([]dto.Job, error) {
	var resp boardResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	jobs := make([]dto.Job, 0, len(resp.JobAdDetails))
	for _, a := range resp.JobAdDetails {
		jobs = append(jobs, dto.Job{
			Title:             a.Title,
			Location:          a.location(),
			URL:               fmt.Sprintf("https://%s.careers.hibob.com/jobs/%s", token, a.ID),
			ProviderPostingID: a.ID,
			Description:       a.fullDescription(),
			SalaryRaw:         a.salaryRaw(),
			WorkArrangement:   a.workArrangement(),
			UpdatedAt:         sources.RFC3339OrNow(a.PublishedAt),
		})
	}
	return jobs, nil
}

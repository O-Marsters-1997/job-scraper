package recruitee

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

// One gate for every company subdomain: Recruitee's limit spans *.recruitee.com.
var limiter = sources.NewGate(2*time.Second, time.Hour)

// New builds a Recruitee source for one company subdomain (e.g. "acmecorp").
func New(token string) *sources.BoardSource {
	return sources.NewBoardSource(sources.BoardSpec{
		Name:        "recruitee",
		URL:         fmt.Sprintf("https://%s.recruitee.com/api/offers/", token),
		CompanySlug: token,
		Parse:       parse,
		Count:       sources.JSONArrayLen("offers"),
		Gate:        limiter,
	})
}

const timestampLayout = "2006-01-02 15:04:05 UTC"

type boardResponse struct {
	Offers []offer `json:"offers"`
}

type offer struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Location    string `json:"location"`
	CareersURL  string `json:"careers_url"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	Remote      bool   `json:"remote"`
	Hybrid      bool   `json:"hybrid"`
	OnSite      bool   `json:"on_site"`
	Salary      struct {
		Min      string `json:"min"`
		Max      string `json:"max"`
		Currency string `json:"currency"`
		Period   string `json:"period"`
	} `json:"salary"`
}

func (o offer) salaryRaw() string {
	low, _ := strconv.Atoi(o.Salary.Min)
	high, _ := strconv.Atoi(o.Salary.Max)
	return sources.SalaryRange(low, high, o.Salary.Currency, o.Salary.Period)
}

func (o offer) workArrangement() string {
	switch {
	case o.Remote:
		return "remote"
	case o.Hybrid:
		return "hybrid"
	case o.OnSite:
		return "onsite"
	default:
		return ""
	}
}

func parse(body []byte) ([]dto.Job, error) {
	var resp boardResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	jobs := make([]dto.Job, 0, len(resp.Offers))
	for _, o := range resp.Offers {
		jobs = append(jobs, dto.Job{
			Title:             o.Title,
			Location:          o.Location,
			URL:               o.CareersURL,
			ProviderPostingID: strconv.FormatInt(o.ID, 10),
			Description:       o.Description,
			SalaryRaw:         o.salaryRaw(),
			WorkArrangement:   o.workArrangement(),
			UpdatedAt:         sources.TimeOrNow(timestampLayout, o.CreatedAt),
		})
	}
	return jobs, nil
}

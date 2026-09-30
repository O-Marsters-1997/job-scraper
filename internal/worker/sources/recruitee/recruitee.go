package recruitee

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

// New builds a Recruitee source for one company subdomain (e.g. "acmecorp").
func New(token string) *sources.BoardSource {
	return sources.NewBoardSource(sources.BoardSpec{
		Name:        "recruitee",
		URL:         fmt.Sprintf("https://%s.recruitee.com/api/offers/", token),
		CompanySlug: token,
		Parse:       parse,
	})
}

type boardResponse struct {
	Offers []offer `json:"offers"`
}

type offer struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Location    string `json:"location"`
	CareersURL  string `json:"careers_url"`
	Description string `json:"description"`
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
			UpdatedAt:         time.Now().UTC(),
		})
	}
	return jobs, nil
}

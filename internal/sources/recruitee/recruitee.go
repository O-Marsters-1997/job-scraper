package recruitee

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

// Config holds Recruitee-specific settings.
type Config struct {
	// Boards is the list of Recruitee company subdomains to scrape (e.g. "acmecorp").
	Boards []string
}

func New(cfg Config) *sources.BoardSource {
	return sources.NewBoardSource(cfg.Boards, sources.BoardSpec{
		Name:      "recruitee",
		URLPrefix: "https://recruitee.com",
		URL: func(token string) string {
			return fmt.Sprintf("https://%s.recruitee.com/api/offers/", token)
		},
		Parse: parse,
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

func parse(body []byte, token string) ([]dto.Job, error) {
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
			CompanySlug:       token,
			ProviderPostingID: strconv.FormatInt(o.ID, 10),
			Source:            "recruitee",
			Description:       o.Description,
			UpdatedAt:         time.Now().UTC(),
		})
	}
	return jobs, nil
}

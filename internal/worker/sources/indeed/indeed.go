package indeed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/slug"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

const (
	apiURL    = "https://apis.indeed.com/graphql"
	jobURL    = "https://uk.indeed.com/viewjob?jk="
	apiKeyEnv = "INDEED_API_KEY"

	resultFields = `pageInfo { nextCursor } results { job { key title datePublished description { html }
location { formatted { long } }
compensation { currencyCode baseSalary { unitOfWork range { ... on Range { min max } } } }
employer { name } } }`
)

var apiHeaders = http.Header{
	"Accept":          {"application/json"},
	"Accept-Language": {"en-GB,en;q=0.9"},
	"User-Agent":      {"Mozilla/5.0 (iPhone; CPU iPhone OS 16_6_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Indeed App 193.1"},
	"Indeed-Co":       {"GB"},
	"Indeed-Locale":   {"en-GB"},
	"Indeed-App-Info": {"appv=193.1; appid=com.indeed.jobsearch; osv=16.6.1; os=ios; dtype=phone"},
}

type Scraper struct {
	sources.PaginatedBase
	keywords string
	filters  map[string]string
	recency  string
}

var _ sources.Source = (*Scraper)(nil)

const recencyMargin = time.Hour

// Recency returns the whole-day window covering the time since lastSucceeded plus an hour, rounded up
// and capped at configured (the Target's own recency filter). It returns "" when the
// Target's stored recency should be used unchanged.
func Recency(configured string, lastSucceeded *time.Time, now time.Time) string {
	if lastSucceeded == nil {
		return ""
	}
	elapsed := max(now.Sub(*lastSucceeded), 0) + recencyMargin
	days := max(int((elapsed+24*time.Hour-1)/(24*time.Hour)), 1)
	if n, err := strconv.Atoi(configured); err == nil && days >= n {
		return ""
	}
	return strconv.Itoa(days)
}

// New builds an Indeed Source. A non-empty recency (from Recency) narrows the window
// and switches to date sort, which makes the Source NewestFirst.
func New(keywords string, filters map[string]string, recency string) *Scraper {
	return &Scraper{
		PaginatedBase: sources.NewBase(sources.Config{
			Name:        "indeed",
			Route:       sources.RouteTiered,
			NewestFirst: recency != "",
		}),
		keywords: keywords,
		filters:  filters,
		recency:  recency,
	}
}

func (s *Scraper) FetchPage(ctx context.Context, cursor string) ([]dto.Job, string, error) {
	key := os.Getenv(apiKeyEnv)
	if key == "" {
		return nil, "", fmt.Errorf("indeed: %s is required", apiKeyEnv)
	}
	payload, err := json.Marshal(map[string]string{"query": s.query(cursor)})
	if err != nil {
		return nil, "", err
	}
	header := apiHeaders.Clone()
	header.Set("Indeed-Api-Key", key)
	body, err := s.PostJSON(ctx, apiURL, payload, header)
	if err != nil {
		var statusErr *sources.StatusError
		if errors.As(err, &statusErr) && (statusErr.Code == http.StatusUnauthorized || statusErr.Code == http.StatusForbidden) {
			return nil, "", fmt.Errorf("indeed: %w: %w", sources.ErrSourceKeyRejected, err)
		}
		return nil, "", fmt.Errorf("indeed: %w", err)
	}
	return parse(body)
}

func (s *Scraper) query(cursor string) string {
	sort := "RELEVANCE"
	if s.recency != "" {
		sort = "DATE"
	}
	args := []string{"limit: 100", "sort: " + sort}
	if s.keywords != "" {
		args = append(args, "what: "+quote(s.keywords))
	}
	if where := s.filters["location"]; where != "" {
		loc := "location: {where: " + quote(where)
		if radius := s.filters["radius"]; radius != "" && sourcespec.ValidFilterValue("indeed", "radius", radius) {
			loc += ", radius: " + radius + ", radiusUnit: MILES"
		}
		args = append(args, loc+"}")
	}
	recency := s.filters["recency"]
	if days, err := strconv.Atoi(s.recency); err == nil {
		args = append(args, windowFilter(days))
	} else if days, err := strconv.Atoi(recency); err == nil && sourcespec.ValidFilterValue("indeed", "recency", recency) {
		args = append(args, windowFilter(days))
	}
	if cursor != "" {
		args = append(args, "cursor: "+quote(cursor))
	}
	return "query { jobSearch(" + strings.Join(args, " ") + ") { " + resultFields + " } }"
}

func windowFilter(days int) string {
	return fmt.Sprintf(`filters: [{ date: { field: "dateOnIndeed", start: "%dh" } }]`, days*24)
}

func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

type response struct {
	Data *struct {
		JobSearch struct {
			PageInfo struct {
				NextCursor string `json:"nextCursor"`
			} `json:"pageInfo"`
			Results []struct {
				Job result `json:"job"`
			} `json:"results"`
		} `json:"jobSearch"`
	} `json:"data"`
	Errors []struct {
		Message    string `json:"message"`
		Extensions struct {
			Code string `json:"code"`
		} `json:"extensions"`
	} `json:"errors"`
}

type result struct {
	Key           string `json:"key"`
	Title         string `json:"title"`
	DatePublished int64  `json:"datePublished"`
	Description   struct {
		HTML string `json:"html"`
	} `json:"description"`
	Location struct {
		Formatted struct {
			Long string `json:"long"`
		} `json:"formatted"`
	} `json:"location"`
	Compensation *struct {
		CurrencyCode string `json:"currencyCode"`
		BaseSalary   *struct {
			UnitOfWork string `json:"unitOfWork"`
			Range      *struct {
				Min *float64 `json:"min"`
				Max *float64 `json:"max"`
			} `json:"range"`
		} `json:"baseSalary"`
	} `json:"compensation"`
	Employer *struct {
		Name string `json:"name"`
	} `json:"employer"`
}

func parse(body []byte) ([]dto.Job, string, error) {
	var resp response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, "", fmt.Errorf("indeed: parse json: %w", err)
	}
	if resp.Data == nil {
		if len(resp.Errors) > 0 {
			switch resp.Errors[0].Extensions.Code {
			case "UNAUTHENTICATED", "UNAUTHORIZED", "FORBIDDEN":
				return nil, "", fmt.Errorf("indeed: %w: %s", sources.ErrSourceKeyRejected, resp.Errors[0].Message)
			}
			return nil, "", fmt.Errorf("indeed: graphql error: %s", resp.Errors[0].Message)
		}
		return nil, "", errors.New("indeed: response has no data")
	}

	search := resp.Data.JobSearch
	jobs := make([]dto.Job, 0, len(search.Results))
	for _, r := range search.Results {
		j := r.Job
		if j.Key == "" {
			continue
		}
		company := ""
		if j.Employer != nil {
			company = j.Employer.Name
		}
		url := jobURL + j.Key
		updatedAt := time.Now().UTC()
		if j.DatePublished > 0 {
			updatedAt = time.UnixMilli(j.DatePublished).UTC()
		} else {
			sources.WarnDefaulted("indeed", "UpdatedAt", url)
		}
		jobs = append(jobs, dto.Job{
			Title:           j.Title,
			Location:        j.Location.Formatted.Long,
			URL:             url,
			CompanySlug:     slug.Company(company),
			Description:     j.Description.HTML,
			SalaryRaw:       formatSalary(j),
			WorkArrangement: sources.DetectWorkArrangement(j.Title + " " + j.Location.Formatted.Long + " " + j.Description.HTML),
			UpdatedAt:       updatedAt,
		})
	}
	return jobs, search.PageInfo.NextCursor, nil
}

func formatSalary(j result) string {
	c := j.Compensation
	if c == nil || c.BaseSalary == nil || c.BaseSalary.Range == nil {
		return ""
	}
	symbol := map[string]string{"GBP": "£", "USD": "$", "EUR": "€"}[c.CurrencyCode]
	if symbol == "" {
		symbol = c.CurrencyCode + " "
	}
	var amounts []string
	for _, v := range []*float64{c.BaseSalary.Range.Min, c.BaseSalary.Range.Max} {
		if v != nil {
			amounts = append(amounts, fmt.Sprintf("%s%.0f", symbol, *v))
		}
	}
	if len(amounts) == 0 {
		return ""
	}
	return strings.Join(amounts, " - ") + " per " + strings.ToLower(c.BaseSalary.UnitOfWork)
}

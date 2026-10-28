package wttj

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/sourcespec"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

const (
	apiURL = "https://api.exp.welcometothejungle.com/graphql"

	maxQueryComplexity = 500
	blockFor           = 5 * time.Minute
)

const (
	roleFields    = `locationPreferences { location } subFunction { value } minYearsExperienceRequired`
	listingFields = roleFields + ` originalUrl actualOriginalUrl company { name urlSafeName }`
	detailFields  = listingFields + ` title firstLiveAt insertedAt involvesBullets { value } requirements { value }`
	companyFields = `name jobLocations mission parsedHqAddress glassdoorRating yearEmployeeGrowthPercentage
sectorTags { value } size { value } visaSponsorshipCountries { location offersVisa }
totalFunding(currency: USD) { amount } fundingRounds(currency: USD) { id }
liveJobs { externalId ` + roleFields + ` }`
)

type Client struct {
	sources.PaginatedBase
	gate *sources.Gate
}

func NewClient(gap time.Duration) *Client {
	return &Client{PaginatedBase: sources.NewBase(sources.Config{Name: name}), gate: sources.NewGate(gap, blockFor)}
}

type Listing struct {
	ID           string
	Locations    []string
	SubFunction  string
	MinYears     *int
	ApplyURL     string
	CompanyName  string
	CompanyToken string
}

const maxMinYears = 5

func eligibleSubFunction(v string) bool {
	return slices.Contains([]string{"Backend Engineer", "Frontend Engineer", "Full-stack Engineer"}, v)
}

func (l Listing) Eligible() bool {
	ukOrRemote := false
	for _, loc := range l.Locations {
		if isUKLocation(loc) || loc == "REMOTE" {
			ukOrRemote = true
		}
	}
	senior := l.MinYears != nil && *l.MinYears > maxMinYears
	return ukOrRemote && eligibleSubFunction(l.SubFunction) && !senior
}

func (c *Client) EligibleListings(ctx context.Context, ids []string) ([]Listing, error) {
	var out []Listing
	var pending []string
	flush := func() error {
		jobs, err := c.jobs(ctx, pending, listingFields)
		for _, j := range jobs {
			out = append(out, j.listing())
		}
		pending = nil
		return err
	}
	size := batchSize(roleFields)
	for start := 0; start < len(ids); start += size {
		roles, err := c.jobs(ctx, ids[start:min(start+size, len(ids))], roleFields)
		if err != nil {
			return out, err
		}
		for _, j := range roles {
			if j.listing().Eligible() {
				pending = append(pending, j.ExternalID)
			}
		}
		if len(pending) >= batchSize(listingFields) {
			if err := flush(); err != nil {
				return out, err
			}
		}
	}
	return out, flush()
}

func batchSize(fields string) int {
	perJob := len(strings.Fields(strings.NewReplacer("{", " ", "}", " ").Replace(fields))) + 1
	return maxQueryComplexity / perJob
}

type valueField struct {
	Value string `json:"value"`
}

type apiJob struct {
	ExternalID        string       `json:"externalId"`
	OriginalURL       string       `json:"originalUrl"`
	ActualOriginalURL string       `json:"actualOriginalUrl"`
	Title             string       `json:"title"`
	FirstLiveAt       string       `json:"firstLiveAt"`
	InsertedAt        string       `json:"insertedAt"`
	SubFunction       *valueField  `json:"subFunction"`
	MinYears          *int         `json:"minYearsExperienceRequired"`
	InvolvesBullets   []valueField `json:"involvesBullets"`
	Requirements      []valueField `json:"requirements"`
	LocationPrefs     []struct {
		Location string `json:"location"`
	} `json:"locationPreferences"`
	Company struct {
		Name        string `json:"name"`
		URLSafeName string `json:"urlSafeName"`
	} `json:"company"`
}

func (j apiJob) listing() Listing {
	l := Listing{
		ID:           j.ExternalID,
		ApplyURL:     cmp.Or(j.ActualOriginalURL, j.OriginalURL),
		CompanyName:  j.Company.Name,
		CompanyToken: j.Company.URLSafeName,
		MinYears:     j.MinYears,
	}
	if j.SubFunction != nil {
		l.SubFunction = j.SubFunction.Value
	}
	for _, p := range j.LocationPrefs {
		l.Locations = append(l.Locations, p.Location)
	}
	return l
}

type apiCompany struct {
	Name          string       `json:"name"`
	JobLocations  []string     `json:"jobLocations"`
	Mission       string       `json:"mission"`
	HQ            string       `json:"parsedHqAddress"`
	Glassdoor     string       `json:"glassdoorRating"`
	Growth        *float64     `json:"yearEmployeeGrowthPercentage"`
	Sectors       []valueField `json:"sectorTags"`
	Size          *valueField  `json:"size"`
	VisaCountries []visa       `json:"visaSponsorshipCountries"`
	TotalFunding  *amount      `json:"totalFunding"`
	FundingRounds []struct{}   `json:"fundingRounds"`
	LiveJobs      []apiJob     `json:"liveJobs"`
}

type visa struct {
	Location   string `json:"location"`
	OffersVisa bool   `json:"offersVisa"`
}

type amount struct {
	Amount string `json:"amount"`
}

func (c apiCompany) inUK() bool { return slices.ContainsFunc(c.JobLocations, isUKLocation) }

func (c *Client) company(ctx context.Context, token string) (apiCompany, error) {
	if !sourcespec.ValidBoardToken(token) {
		return apiCompany{}, fmt.Errorf("wttj: invalid company token %q", token)
	}
	var data struct {
		Company *apiCompany `json:"company"`
	}
	if err := c.query(ctx, fmt.Sprintf("query { company(urlSafeName: %q) { %s } }", token, companyFields), &data); err != nil {
		return apiCompany{}, err
	}
	if data.Company == nil {
		return apiCompany{}, fmt.Errorf("%w: wttj company %s", sources.ErrGone, token)
	}
	return *data.Company, nil
}

func (c *Client) jobs(ctx context.Context, ids []string, fields string) ([]apiJob, error) {
	var out []apiJob
	size := batchSize(fields)
	for start := 0; start < len(ids); start += size {
		batch := ids[start:min(start+size, len(ids))]
		var q strings.Builder
		q.WriteString("query {")
		for i, id := range batch {
			if !sourcespec.ValidBoardToken(id) {
				return out, fmt.Errorf("wttj: invalid job id %q", id)
			}
			fmt.Fprintf(&q, " j%d: publicJob(externalId: %q) { ...J }", i, id)
		}
		fmt.Fprintf(&q, " } fragment J on Job { %s }", fields)
		var data map[string]*apiJob
		if err := c.query(ctx, q.String(), &data); err != nil {
			return out, err
		}
		for i, id := range batch {
			if j := data[fmt.Sprintf("j%d", i)]; j != nil {
				j.ExternalID = id
				out = append(out, *j)
			}
		}
	}
	return out, nil
}

func (c *Client) query(ctx context.Context, query string, data any) error {
	if err := c.gate.Wait(ctx); err != nil {
		return err
	}
	body, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		return fmt.Errorf("wttj: marshal query: %w", err)
	}
	resp, err := c.PostJSON(ctx, apiURL, body, http.Header{"Accept": {"application/json"}})
	var status *sources.StatusError
	if errors.As(err, &status) && (status.Code == http.StatusForbidden || status.Code == http.StatusTooManyRequests) {
		until := c.gate.Block(status.RetryAfter)
		return fmt.Errorf("%w: %w", &sources.DeferredError{Until: until}, err)
	}
	if err != nil {
		return err
	}
	var r struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(resp, &r); err != nil {
		return fmt.Errorf("%w: wttj: decode response: %w", sources.ErrUnparseable, err)
	}
	if len(r.Data) == 0 || string(r.Data) == "null" {
		msg := "no data"
		if len(r.Errors) > 0 {
			msg = r.Errors[0].Message
		}
		return fmt.Errorf("%w: wttj: %s", sources.ErrUnparseable, msg)
	}
	if err := json.Unmarshal(r.Data, data); err != nil {
		return fmt.Errorf("%w: wttj: decode data: %w", sources.ErrUnparseable, err)
	}
	return nil
}

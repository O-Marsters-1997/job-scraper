package wttj

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/slug"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

const (
	name    = "wttj"
	baseURL = "https://app.welcometothejungle.com"
	maxAge  = 90 * 24 * time.Hour

	ukPollIn    = 7 * 24 * time.Hour
	otherPollIn = 90 * 24 * time.Hour
	blockFor    = 24 * time.Hour
)

var (
	apolloRe = regexp.MustCompile(`__APOLLO_STATE__=__b64dec\("([^"]+)"\)`)
	ldJSONRe = regexp.MustCompile(`(?s)<script[^>]*type="application/ld\+json"[^>]*>(.*?)</script>`)

	ukLocations = map[string]bool{
		"LONDON": true, "REMOTE_UK": true, "MANCHESTER": true, "EDINBURGH": true, "BRISTOL": true,
		"CAMBRIDGE": true, "OXFORD": true, "BIRMINGHAM": true, "LEEDS": true, "GLASGOW": true,
		"BELFAST": true, "CARDIFF": true,
	}

	postings = &postingCache{byToken: map[string]map[string]*dto.Job{}}
	limiter  = sources.NewGate(2*time.Second, blockFor)
)

// postingCache remembers, per Board token, every job page already fetched, so a repeat poll
// re-reads only the company page. A nil entry is a page that failed the age gate.
type postingCache struct {
	mu      sync.Mutex
	byToken map[string]map[string]*dto.Job
}

func (c *postingCache) get(token string) map[string]*dto.Job {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.byToken[token]
}

func (c *postingCache) put(token string, jobs map[string]*dto.Job) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.byToken[token] = jobs
}

// Source polls one WTTJ company: the Board token is the company's urlSafeName.
type Source struct {
	sources.PaginatedBase
	token string
	now   func() time.Time
}

var _ sources.Source = (*Source)(nil)

func New(token string, now func() time.Time) *Source {
	return &Source{PaginatedBase: sources.NewBase(sources.Config{Name: name}), token: token, now: now}
}

func (s *Source) FetchPage(ctx context.Context, cursor string) ([]dto.Job, string, error) {
	if cursor != "" {
		return nil, "", fmt.Errorf("%s: unexpected cursor %q", name, cursor)
	}
	res, err := s.PollBoard(ctx)
	return res.Jobs, "", err
}

func (s *Source) get(ctx context.Context, url string) ([]byte, error) {
	if err := limiter.Wait(ctx); err != nil {
		return nil, err
	}
	body, err := s.Get(ctx, url)
	var statusErr *sources.StatusError
	if errors.As(err, &statusErr) && (statusErr.Code == http.StatusForbidden || statusErr.Code == http.StatusTooManyRequests) {
		limiter.Block(blockFor)
	}
	return body, err
}

func (s *Source) PollBoard(ctx context.Context) (sources.BoardResult, error) {
	res, _, err := s.poll(ctx)
	return res, err
}

// DiscoverTimeout fits a company page plus its eligible job pages at the 2s request gap.
const DiscoverTimeout = 3 * time.Minute

// Discovery is what a first look at a company yields: its real name, its Jobs, and
// whether the company is in the UK.
type Discovery struct {
	Name    string
	Jobs    []dto.Job
	UK      bool
	Profile *dto.CompanyProfile
}

func (s *Source) Discover(ctx context.Context) (Discovery, error) {
	res, company, err := s.poll(ctx)
	if err != nil {
		return Discovery{}, err
	}
	return Discovery{Name: company.Name, Jobs: res.Jobs, UK: company.inUK(), Profile: company.Profile}, nil
}

func (s *Source) poll(ctx context.Context) (sources.BoardResult, company, error) {
	body, err := s.get(ctx, baseURL+"/companies/"+url.PathEscape(s.token))
	if err != nil {
		return sources.BoardResult{}, company{}, err
	}
	company, err := parseCompanyState(body)
	if err != nil {
		return sources.BoardResult{}, company, err
	}
	if !company.inUK() {
		return sources.BoardResult{NextPollIn: otherPollIn, Profile: company.Profile}, company, nil
	}

	known := postings.get(s.token)
	seen := make(map[string]*dto.Job, len(company.Jobs))
	var jobs []dto.Job
	for _, cj := range company.Jobs {
		if !cj.eligible() {
			continue
		}
		job, ok := known[cj.ID]
		if !ok {
			job, err = s.fetchPosting(ctx, cj.ID)
			if err != nil {
				return sources.BoardResult{}, company, err
			}
			if job != nil && s.now().Sub(job.UpdatedAt) > maxAge {
				job = nil
			}
		}
		seen[cj.ID] = job
		if job != nil {
			jobs = append(jobs, *job)
		}
	}
	postings.put(s.token, seen)
	return sources.BoardResult{Jobs: jobs, NextPollIn: ukPollIn, Profile: company.Profile}, company, nil
}

// fetchPosting returns nil for a job whose page is gone or unparseable.
func (s *Source) fetchPosting(ctx context.Context, id string) (*dto.Job, error) {
	page, err := s.get(ctx, baseURL+"/jobs/"+url.PathEscape(id))
	if errors.Is(err, sources.ErrGone) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	job, err := parseJobPosting(page, id)
	if err != nil {
		slog.WarnContext(ctx, "wttj job page unparseable", slog.String("job_id", id), slog.Any(logger.KeyErr, err))
		return nil, nil
	}
	return &job, nil
}

type company struct {
	Name         string
	JobLocations []string
	Jobs         []companyJob
	Profile      *dto.CompanyProfile
}

type companyJob struct {
	ID        string
	Title     string
	Locations []string
	Function  string
}

func (c company) inUK() bool {
	for _, l := range c.JobLocations {
		if isUKLocation(l) {
			return true
		}
	}
	return false
}

func (j companyJob) eligible() bool {
	ukOrRemote := false
	for _, l := range j.Locations {
		if isUKLocation(l) || l == "REMOTE" {
			ukOrRemote = true
		}
	}
	return ukOrRemote && strings.Contains(j.Function, "Engineering")
}

func isUKLocation(code string) bool { return ukLocations[code] || strings.HasSuffix(code, "_UK") }

type apolloEntity struct {
	Typename      string      `json:"__typename"`
	ID            string      `json:"id"`
	ExternalID    string      `json:"externalId"`
	Name          string      `json:"name"`
	Title         string      `json:"title"`
	Value         string      `json:"value"`
	Location      string      `json:"location"`
	JobLocations  []string    `json:"jobLocations"`
	LiveJobs      []apolloRef `json:"liveJobs"`
	LocationPrefs []apolloRef `json:"locationPreferences"`
	Function      *apolloRef  `json:"function"`
}

type apolloRef struct {
	Ref string `json:"__ref"`
}

func parseCompanyState(body []byte) (company, error) {
	m := apolloRe.FindSubmatch(body)
	if m == nil {
		return company{}, errors.New("wttj: no __APOLLO_STATE__ in company page")
	}
	raw, err := base64.StdEncoding.DecodeString(string(m[1]))
	if err != nil {
		return company{}, fmt.Errorf("wttj: decode state: %w", err)
	}
	var state map[string]apolloEntity
	if err := json.Unmarshal(raw, &state); err != nil {
		return company{}, fmt.Errorf("wttj: parse state: %w", err)
	}
	var rawState map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawState); err != nil {
		return company{}, fmt.Errorf("wttj: parse state: %w", err)
	}
	for key, e := range state {
		if e.Typename != "Company" {
			continue
		}
		c := company{Name: e.Name, JobLocations: e.JobLocations, Profile: parseProfile(rawState, key)}
		for _, ref := range e.LiveJobs {
			j := state[ref.Ref]
			cj := companyJob{ID: j.ExternalID, Title: j.Title}
			for _, lp := range j.LocationPrefs {
				cj.Locations = append(cj.Locations, state[lp.Ref].Location)
			}
			if j.Function != nil {
				cj.Function = state[j.Function.Ref].Value
			}
			c.Jobs = append(c.Jobs, cj)
		}
		return c, nil
	}
	return company{}, errors.New("wttj: no Company in state")
}

type jobPosting struct {
	Title            string `json:"title"`
	DatePosted       string `json:"datePosted"`
	Description      string `json:"description"`
	Responsibilities string `json:"responsibilities"`
	Skills           string `json:"skills"`
	HiringOrg        struct {
		Name string `json:"name"`
	} `json:"hiringOrganization"`
	Identifier struct {
		Value string `json:"value"`
	} `json:"identifier"`
	Locations []struct {
		Address struct {
			Locality string `json:"addressLocality"`
			Country  string `json:"addressCountry"`
		} `json:"address"`
	} `json:"jobLocation"`
}

func parseJobPosting(body []byte, id string) (dto.Job, error) {
	m := ldJSONRe.FindSubmatch(body)
	if m == nil {
		return dto.Job{}, errors.New("wttj: no JSON-LD in job page")
	}
	var p jobPosting
	if err := json.Unmarshal(m[1], &p); err != nil {
		return dto.Job{}, fmt.Errorf("wttj: parse JSON-LD: %w", err)
	}
	posted, err := time.Parse("2006-01-02T15:04:05", p.DatePosted)
	if err != nil {
		return dto.Job{}, fmt.Errorf("wttj: datePosted %q: %w", p.DatePosted, err)
	}
	places := make([]string, 0, len(p.Locations))
	for _, l := range p.Locations {
		places = append(places, strings.Trim(l.Address.Locality+", "+l.Address.Country, ", "))
	}
	return dto.Job{
		Title:             strings.TrimSpace(p.Title),
		Location:          strings.Join(places, "; "),
		URL:               baseURL + "/jobs/" + id,
		ApplyURL:          p.Identifier.Value,
		CompanySlug:       slug.Make(p.HiringOrg.Name),
		ProviderPostingID: id,
		Source:            name,
		UpdatedAt:         posted,
		Description:       strings.Join([]string{p.Responsibilities, p.Description, p.Skills}, "\n"),
	}, nil
}

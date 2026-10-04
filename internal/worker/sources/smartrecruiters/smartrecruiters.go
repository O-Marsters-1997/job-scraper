package smartrecruiters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

const (
	name          = "smartrecruiters"
	apiBase       = "https://api.smartrecruiters.com/v1/companies/"
	pageSize      = 100
	maxPages      = 200
	detailWorkers = 8
)

var postings = &postingCache{byToken: map[string]map[string]dto.Job{}}

// postingCache remembers, per Board token, every posting already read in full, keyed by id and
// release date, so a repeat poll fetches details only for new or re-released postings.
type postingCache struct {
	mu      sync.Mutex
	byToken map[string]map[string]dto.Job
}

func (c *postingCache) get(token string) map[string]dto.Job {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.byToken[token]
}

func (c *postingCache) put(token string, jobs map[string]dto.Job) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.byToken[token] = jobs
}

// Source polls one SmartRecruiters company. The Board token is the company identifier, whose
// case the API preserves in URLs (e.g. "Wise").
type Source struct {
	sources.PaginatedBase
	token string
}

var (
	_ sources.Source      = (*Source)(nil)
	_ sources.BoardPoller = (*Source)(nil)
)

func New(token string) *Source {
	return &Source{PaginatedBase: sources.NewBase(sources.Config{Name: name}), token: token}
}

func (s *Source) FetchPage(ctx context.Context, cursor string) ([]dto.Job, string, error) {
	if cursor != "" {
		return nil, "", fmt.Errorf("%s: unexpected cursor %q", name, cursor)
	}
	res, err := s.PollBoard(ctx)
	return res.Jobs, "", err
}

type listResponse struct {
	TotalFound int       `json:"totalFound"`
	Content    []listing `json:"content"`
}

type listing struct {
	ID           string `json:"id"`
	ReleasedDate string `json:"releasedDate"`
	Company      struct {
		Name string `json:"name"`
	} `json:"company"`
}

func (l listing) cacheKey() string { return l.ID + "@" + l.ReleasedDate }

// PollBoard lists every posting, then reads in full only those not already cached. Reported is
// the API's totalFound.
func (s *Source) PollBoard(ctx context.Context) (sources.BoardResult, error) {
	listed, total, err := s.list(ctx)
	if err != nil {
		return sources.BoardResult{}, err
	}

	known := postings.get(s.token)
	current := make(map[string]dto.Job, len(listed))
	var missing []listing
	for _, l := range listed {
		key := l.cacheKey()
		if job, ok := known[key]; ok {
			current[key] = job
		} else {
			missing = append(missing, l)
		}
	}

	fetched, err := s.fetchDetails(ctx, missing)
	if err != nil {
		postings.put(s.token, mergeMaps(known, fetched))
		return sources.BoardResult{}, err
	}
	maps.Copy(current, fetched)
	postings.put(s.token, current)

	jobs := make([]dto.Job, 0, len(current))
	for _, l := range listed {
		if job, ok := current[l.cacheKey()]; ok {
			jobs = append(jobs, job)
		}
	}
	return sources.BoardResult{Jobs: jobs, Reported: total}, nil
}

func mergeMaps(a, b map[string]dto.Job) map[string]dto.Job {
	merged := make(map[string]dto.Job, len(a)+len(b))
	maps.Copy(merged, a)
	maps.Copy(merged, b)
	return merged
}

func (s *Source) list(ctx context.Context) ([]listing, int, error) {
	var all []listing
	total := 0
	for page := 0; ; page++ {
		if page >= maxPages {
			return nil, 0, fmt.Errorf("%s: board exceeds %d pages", name, maxPages)
		}
		body, err := s.Get(ctx, fmt.Sprintf("%s%s/postings?limit=%d&offset=%d", apiBase, url.PathEscape(s.token), pageSize, len(all)))
		if err != nil {
			return nil, 0, fmt.Errorf("%s: list page %d: %w", name, page+1, err)
		}
		var resp listResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, 0, fmt.Errorf("%s: parse list: %w", name, err)
		}
		total = resp.TotalFound
		all = append(all, resp.Content...)
		if len(resp.Content) == 0 || len(all) >= total {
			return all, total, nil
		}
	}
}

// fetchDetails reads each listing's posting concurrently. It returns whatever it fetched even on
// error, so a retry does not repeat finished work. A posting withdrawn since the list is skipped.
func (s *Source) fetchDetails(ctx context.Context, missing []listing) (map[string]dto.Job, error) {
	var (
		mu      sync.Mutex
		fetched = make(map[string]dto.Job, len(missing))
		group   errgroup.Group
	)
	group.SetLimit(detailWorkers)
	for _, l := range missing {
		group.Go(func() error {
			body, err := s.Get(ctx, fmt.Sprintf("%s%s/postings/%s", apiBase, url.PathEscape(s.token), url.PathEscape(l.ID)))
			if errors.Is(err, sources.ErrGone) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("%s: posting %s: %w", name, l.ID, err)
			}
			job, err := s.parseDetail(body)
			if err != nil {
				return fmt.Errorf("%s: posting %s: %w", name, l.ID, err)
			}
			mu.Lock()
			fetched[l.cacheKey()] = job
			mu.Unlock()
			return nil
		})
	}
	return fetched, group.Wait()
}

type detail struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	PostingURL   string `json:"postingUrl"`
	ReleasedDate string `json:"releasedDate"`
	Location     struct {
		FullLocation string `json:"fullLocation"`
		Remote       bool   `json:"remote"`
		Hybrid       bool   `json:"hybrid"`
	} `json:"location"`
	Compensation *struct {
		Min      float64 `json:"min"`
		Max      float64 `json:"max"`
		Currency string  `json:"currency"`
		Period   string  `json:"period"`
	} `json:"compensation"`
	JobAd struct {
		Sections struct {
			JobDescription        section `json:"jobDescription"`
			Qualifications        section `json:"qualifications"`
			AdditionalInformation section `json:"additionalInformation"`
			CompanyDescription    section `json:"companyDescription"`
		} `json:"sections"`
	} `json:"jobAd"`
}

type section struct {
	Text string `json:"text"`
}

var salaryPeriods = map[string]string{
	"YEARLY":  "year",
	"MONTHLY": "month",
	"WEEKLY":  "week",
	"DAILY":   "day",
	"HOURLY":  "hour",
}

func (d detail) salaryRaw() string {
	if d.Compensation == nil {
		return ""
	}
	c := d.Compensation
	return sources.SalaryRange(int(c.Min), int(c.Max), c.Currency, salaryPeriods[c.Period])
}

func (d detail) workArrangement() string {
	switch {
	case d.Location.Remote:
		return "remote"
	case d.Location.Hybrid:
		return "hybrid"
	default:
		return "onsite"
	}
}

func (d detail) location() string {
	var parts []string
	for part := range strings.SplitSeq(d.Location.FullLocation, ",") {
		if part = strings.TrimSpace(part); part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, ", ")
}

func (d detail) description() string {
	s := d.JobAd.Sections
	var parts []string
	for _, sec := range []section{s.JobDescription, s.Qualifications, s.AdditionalInformation, s.CompanyDescription} {
		if sec.Text != "" {
			parts = append(parts, sec.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func (s *Source) parseDetail(body []byte) (dto.Job, error) {
	var d detail
	if err := json.Unmarshal(body, &d); err != nil {
		return dto.Job{}, fmt.Errorf("parse json: %w", err)
	}
	postingURL := d.PostingURL
	if postingURL == "" {
		postingURL = fmt.Sprintf("https://jobs.smartrecruiters.com/%s/%s", s.token, d.ID)
	}
	return dto.Job{
		Title:             d.Name,
		Location:          d.location(),
		URL:               postingURL,
		CompanySlug:       s.token,
		ProviderPostingID: d.ID,
		Source:            name,
		Description:       d.description(),
		SalaryRaw:         d.salaryRaw(),
		WorkArrangement:   d.workArrangement(),
		UpdatedAt:         sources.RFC3339OrNow(d.ReleasedDate),
	}, nil
}

// BoardName reads the company's display name from a postings list response, or "" when the
// Board has no postings or the body is not a list.
func BoardName(body []byte) string {
	var resp listResponse
	if json.Unmarshal(body, &resp) != nil || len(resp.Content) == 0 {
		return ""
	}
	return strings.TrimSpace(resp.Content[0].Company.Name)
}

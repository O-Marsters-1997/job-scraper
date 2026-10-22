package wttj

import (
	"cmp"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/slug"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

const (
	name    = "wttj"
	baseURL = "https://app.welcometothejungle.com"
	maxAge  = 90 * 24 * time.Hour

	otherPollIn = 90 * 24 * time.Hour
)

var ukLocations = map[string]bool{
	"LONDON": true, "REMOTE_UK": true, "MANCHESTER": true, "EDINBURGH": true, "BRISTOL": true,
	"CAMBRIDGE": true, "OXFORD": true, "BIRMINGHAM": true, "LEEDS": true, "GLASGOW": true,
	"BELFAST": true, "CARDIFF": true,
}

func isUKLocation(code string) bool { return ukLocations[code] || strings.HasSuffix(code, "_UK") }

type Source struct {
	companyToken string
	now          func() time.Time
	client       *Client
}

var (
	_ sources.Source      = (*Source)(nil)
	_ sources.BoardPoller = (*Source)(nil)
)

func New(companyToken string, now func() time.Time, client *Client) *Source {
	return &Source{companyToken: companyToken, now: now, client: client}
}

func (s *Source) Cfg() sources.Config { return sources.Config{Name: name} }

func (s *Source) FetchPage(ctx context.Context, cursor string) ([]dto.Job, string, error) {
	if cursor != "" {
		return nil, "", fmt.Errorf("%s: unexpected cursor %q", name, cursor)
	}
	res, err := s.PollBoard(ctx)
	return res.Jobs, "", err
}

func (s *Source) PollBoard(ctx context.Context) (sources.BoardResult, error) {
	res, _, err := s.poll(ctx)
	return res, err
}

const DiscoverTimeout = time.Minute

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
	return Discovery{Name: company.Name, Jobs: res.Jobs, UK: company.inUK(), Profile: res.Profile}, nil
}

func (s *Source) poll(ctx context.Context) (sources.BoardResult, apiCompany, error) {
	company, err := s.client.company(ctx, s.companyToken)
	if err != nil {
		return sources.BoardResult{}, company, err
	}
	profile := company.profile()
	if !company.inUK() {
		return sources.BoardResult{NextPollIn: otherPollIn, Profile: profile}, company, nil
	}
	var ids []string
	for _, j := range company.LiveJobs {
		if j.listing().Eligible() {
			ids = append(ids, j.ExternalID)
		}
	}
	details, err := s.client.jobs(ctx, ids, detailFields)
	if err != nil {
		return sources.BoardResult{}, company, err
	}
	var jobs []dto.Job
	for _, d := range details {
		job := d.toJob()
		if job.PostedAt != nil && s.now().Sub(*job.PostedAt) > maxAge {
			continue
		}
		jobs = append(jobs, job)
	}
	return sources.BoardResult{Jobs: jobs, Profile: profile}, company, nil
}

func CompanyName(ctx context.Context, companyToken string) (string, error) {
	c, err := NewClient(0).company(ctx, companyToken)
	return c.Name, err
}

func (j apiJob) toJob() dto.Job {
	l := j.listing()
	places := make([]string, len(l.Locations))
	for i, code := range l.Locations {
		places[i] = placeName(code)
	}
	job := dto.Job{
		Title:             strings.TrimSpace(j.Title),
		Location:          strings.Join(places, "; "),
		URL:               baseURL + "/jobs/" + j.ExternalID,
		ApplyURL:          l.ApplyURL,
		CompanySlug:       slug.Company(j.Company.Name),
		ProviderPostingID: j.ExternalID,
		Source:            name,
		Description:       strings.TrimSpace(bullets(j.InvolvesBullets) + "\n\n" + bullets(j.Requirements)),
	}
	if posted, err := time.Parse("2006-01-02T15:04:05", cmp.Or(j.FirstLiveAt, j.InsertedAt)); err == nil {
		job.PostedAt = &posted
	}
	return job
}

func bullets(items []valueField) string {
	lines := make([]string, len(items))
	for i, item := range items {
		lines[i] = "- " + item.Value
	}
	return strings.Join(lines, "\n")
}

func placeName(code string) string {
	words := strings.Split(strings.ToLower(code), "_")
	country := ""
	if n := len(words); n > 1 && len(words[n-1]) == 2 {
		country, words = strings.ToUpper(words[n-1]), words[:n-1]
	}
	if country == "" && ukLocations[code] {
		country = "UK"
	}
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	place := strings.Join(words, " ")
	if country == "" {
		return place
	}
	return place + ", " + country
}

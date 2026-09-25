// Package indeed scrapes Indeed search-result and job-detail pages.
//
// Indeed sits behind aggressive anti-bot protection (Cloudflare + its own
// fingerprinting), so this source runs through BrightData Web Unlocker
// (UseProxy: true) rather than direct HTTP.
//
// Selector provenance: this environment could not reach Indeed live (no
// BRIGHTDATA_PROXY_URL configured here, and a direct unproxied fetch is
// blocked by a Cloudflare interstitial — see testdata/blocked_cloudflare.html
// for the real captured block page). The selectors below are therefore NOT
// verified against a live Indeed response. They follow the data-testid hooks
// documented by long-running public Indeed scrapers (e.g. the JobSpy project,
// github.com/speedyapply/JobSpy) as the most stable identifiers Indeed has
// exposed in recent years: `[data-testid="company-name"]`,
// `[data-testid="text-location"]` on cards, and
// `[data-testid="inlineHeader-companyName"]` /
// `[data-testid="inlineHeader-companyLocation"]` / `#jobDescriptionText` on
// detail pages.
//
// snapshots/*.html are placeholder fixtures built to match these selectors —
// NOT real captures. Regenerate against real pages once proxy access is
// available:
//
//	just cli download indeed list_search <a-real-indeed-search-url>
//	just cli download indeed detail_job <a-real-indeed-detail-url>
//	just cli rebase indeed
package indeed

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/slug"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

const (
	baseURL = "https://www.indeed.com"

	selJobCard      = `div.job_seen_beacon`
	selCardTitle    = `h2.jobTitle a`
	selCardCompany  = `[data-testid="company-name"]`
	selCardLocation = `[data-testid="text-location"]`

	selDetailTitle       = `h1[data-testid="jobsearch-JobInfoHeader-title"]`
	selDetailCompany     = `[data-testid="inlineHeader-companyName"]`
	selDetailLocation    = `[data-testid="inlineHeader-companyLocation"]`
	selDetailDescription = `#jobDescriptionText`
)

// noResultsRe matches Indeed's plain-text "nothing found" copy. Unverified
// (see package doc) — refine once a real empty-search fixture is captured.
var noResultsRe = regexp.MustCompile(`(?i)did not match any jobs|no jobs found|couldn't find any jobs`)

type Config struct {
	// URLs are the full Indeed search-result URLs configured per source target
	// (kindURL in the registry — each target's value is already a complete
	// search URL, not a token to build one from).
	URLs []string
}

type Scraper struct {
	sources.PaginatedBase
	urls []string
}

var _ sources.Source = (*Scraper)(nil)
var _ sources.DetailFetcher = (*Scraper)(nil)
var _ sources.SnapshotSource = (*Scraper)(nil)

func New(cfg Config) *Scraper {
	return &Scraper{
		PaginatedBase: sources.NewBase(sources.Config{
			Name:     "indeed",
			UseProxy: true,
		}),
		urls: cfg.URLs,
	}
}

// Iterate fetches each configured search URL once.
//
// ponytail: no offset pagination (`&start=N`) — Indeed's total-result-count
// element isn't verified without a live/proxied fetch, so PaginatedBase's
// count-driven IteratePages isn't wired up. Add it once a real search-page
// fixture confirms the count selector.
func (s *Scraper) Iterate(ctx context.Context, fn func(context.Context, []dto.Job) (bool, error)) error {
	for _, searchURL := range s.urls {
		body, err := s.Get(ctx, searchURL)
		if err != nil {
			return fmt.Errorf("indeed: fetch %s: %w", searchURL, err)
		}

		jobs, err := ParseURLs(bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("indeed: parse %s: %w", searchURL, err)
		}

		stop, err := fn(ctx, jobs)
		if err != nil || stop {
			return err
		}
	}
	return nil
}

func (s *Scraper) GetDetails(ctx context.Context, url string) (dto.Job, error) {
	body, err := s.Get(ctx, url)
	if err != nil {
		return dto.Job{}, err
	}
	job, err := ParseJobDetail(bytes.NewReader(body), url)
	if err != nil {
		return dto.Job{}, err
	}
	// Indeed's detail page exposes no reliable machine-readable post date, so
	// UpdatedAt is stamped here rather than in the pure parser — keeps the
	// parser deterministic for snapshot testing (same approach as linkedin).
	if job.UpdatedAt.IsZero() {
		sources.WarnDefaulted("indeed", "UpdatedAt", url)
		job.UpdatedAt = time.Now().UTC()
	}
	return job, nil
}

func (s *Scraper) ParseURLs(r io.Reader) ([]dto.Job, error) { return ParseURLs(r) }

func (s *Scraper) ParseJobDetail(r io.Reader, url string) (dto.Job, error) {
	return ParseJobDetail(r, url)
}

// ParseURLs extracts partial job cards (Title, Location, URL, CompanySlug)
// from an Indeed search-results page.
//
// BrightData's Web Unlocker returns HTTP 200 even for anti-bot interstitials,
// so a page with zero job cards and no recognizable "no results" marker is
// treated as a block and returns an error rather than an empty, falsely-fresh
// result (see testdata/blocked_cloudflare.html + the accompanying test).
func ParseURLs(r io.Reader) ([]dto.Job, error) {
	doc, err := sources.ParseHTML(r)
	if err != nil {
		return nil, err
	}

	var jobs []dto.Job
	doc.Find(selJobCard).Each(func(_ int, card *goquery.Selection) {
		linkEl := card.Find(selCardTitle).First()
		jobURL := cardJobURL(linkEl)
		if jobURL == "" {
			return
		}

		title := strings.TrimSpace(linkEl.Text())
		company := strings.TrimSpace(card.Find(selCardCompany).First().Text())
		location := strings.TrimSpace(card.Find(selCardLocation).First().Text())

		jobs = append(jobs, dto.Job{
			Title:       title,
			Location:    location,
			URL:         jobURL,
			CompanySlug: slug.Make(company),
		})
	})

	if len(jobs) == 0 && !noResultsRe.MatchString(doc.Text()) {
		return nil, fmt.Errorf("indeed: zero job cards and no recognizable no-results marker (possible anti-bot block)")
	}
	return jobs, nil
}

// cardJobURL prefers the canonical /viewjob?jk= form built from data-jk over
// the card's raw href, which is often a /rc/clk redirect/tracking link rather
// than the stable job page itself.
func cardJobURL(linkEl *goquery.Selection) string {
	if jk, ok := linkEl.Attr("data-jk"); ok && jk != "" {
		return baseURL + "/viewjob?jk=" + jk
	}
	href, ok := linkEl.Attr("href")
	if !ok || href == "" {
		return ""
	}
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	return baseURL + href
}

func ParseJobDetail(r io.Reader, url string) (dto.Job, error) {
	doc, err := sources.ParseHTML(r)
	if err != nil {
		return dto.Job{}, err
	}

	title := strings.TrimSpace(doc.Find(selDetailTitle).First().Text())
	if title == "" {
		return dto.Job{}, fmt.Errorf("title not found (selector: %q)", selDetailTitle)
	}

	company := strings.TrimSpace(doc.Find(selDetailCompany).First().Text())
	location := strings.TrimSpace(doc.Find(selDetailLocation).First().Text())

	descNode := doc.Find(selDetailDescription).First()
	descHTML, _ := descNode.Html()
	description := strings.TrimSpace(descHTML)
	descText := descNode.Text()
	if description == "" {
		sources.WarnDefaulted("indeed", "Description", url)
	}

	salaryRaw := sources.ParseSalaryRaw(descText)
	workArrangement := sources.DetectWorkArrangement(title + " " + location + " " + descText)

	return dto.Job{
		Title:           title,
		Location:        location,
		URL:             url,
		CompanySlug:     slug.Make(company),
		Source:          "indeed",
		Description:     description,
		SalaryRaw:       salaryRaw,
		WorkArrangement: workArrangement,
	}, nil
}

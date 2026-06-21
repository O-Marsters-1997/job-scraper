package wis

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/proxy"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

const (
	baseURL        = "https://workinstartups.com/search"
	perPage        = 50
	resultsPerPage = 50

	selJobCard    = `div[data-aid]`
	selJobLink    = `h2 a`
	selTotalCount = `span[data-cy-count]`
	selTitle      = `h1`
	selCompany    = `.ui-company`
	selLocation   = `.ui-location`
	selTime       = `time[datetime]`
)

// Search holds a single configured search: a keyword query and an optional region.
type Search struct {
	Keywords string // maps to the q URL param, e.g. "product engineer"
	Region   string // maps to the w URL param, e.g. "uk"; empty means no filter
}

func (s Search) startURL() string {
	v := url.Values{}
	v.Set("q", s.Keywords)
	if s.Region != "" {
		v.Set("w", s.Region)
	}
	v.Set("per_page", strconv.Itoa(perPage))
	return baseURL + "?" + v.Encode()
}

func pageURL(s Search, page int) string {
	if page <= 1 {
		return s.startURL()
	}
	return fmt.Sprintf("%s&p=%d", s.startURL(), page)
}

// Config holds all user-configured searches for this source.
type Config struct {
	Searches []Search
}

type Scraper struct {
	sources.PaginatedBase
	searches []Search
}

var _ sources.Source = (*Scraper)(nil)
var _ sources.DetailFetcher = (*Scraper)(nil)

func New(cfg Config) *Scraper {
	return &Scraper{
		PaginatedBase: sources.NewBase(sources.Config{
			Name:              "wis",
			URLPrefix:         "https://workinstartups.com",
			Schedule:          "0 */6 * * *",
			MinScrapeInterval: 5 * time.Hour,
			ProxyTier:         proxy.Datacenter,
		}),
		searches: cfg.Searches,
	}
}

func (s *Scraper) fetchPage(ctx context.Context, search Search, page int) (urls []string, totalCount int, err error) {
	body, err := s.Get(ctx, pageURL(search, page))
	if err != nil {
		return nil, 0, err
	}

	jobs, err := ParseURLs(bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	urls = make([]string, len(jobs))
	for i, j := range jobs {
		urls[i] = j.URL
	}

	if page == 1 {
		totalCount, err = ParseTotalCount(bytes.NewReader(body))
		if err != nil {
			return nil, 0, err
		}
	}

	return urls, totalCount, nil
}

func (s *Scraper) ParseURLs(r io.Reader) ([]dto.Job, error) {
	return ParseURLs(r)
}

func (s *Scraper) ParseJobDetail(r io.Reader, url string) (dto.Job, error) {
	return ParseJobDetail(r, url)
}

func (s *Scraper) Iterate(ctx context.Context, fn func(context.Context, []dto.Job) (bool, error)) error {
	for _, search := range s.searches {
		fetch := func(ctx context.Context, page int) ([]string, int, error) {
			return s.fetchPage(ctx, search, page)
		}
		err := s.IteratePages(ctx, func(ctx context.Context, urls []string) (bool, error) {
			jobs := make([]dto.Job, len(urls))
			for i, u := range urls {
				jobs[i] = dto.Job{URL: u}
			}
			return fn(ctx, jobs)
		}, fetch, resultsPerPage)
		if err != nil {
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
	return ParseJobDetail(bytes.NewReader(body), url)
}

func ParseTotalCount(r io.Reader) (int, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return 0, fmt.Errorf("parse html: %w", err)
	}
	val := doc.Find(selTotalCount).First().AttrOr("data-cy-count", "")
	if val == "" {
		return 0, fmt.Errorf("data-cy-count element not found")
	}
	count, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("parse data-cy-count: %w", err)
	}
	return count, nil
}

func TotalPages(totalCount int) int {
	return (totalCount + resultsPerPage - 1) / resultsPerPage
}

func ParseURLs(r io.Reader) ([]dto.Job, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}
	var jobs []dto.Job
	doc.Find(selJobCard).Each(func(_ int, card *goquery.Selection) {
		linkEl := card.Find(selJobLink)
		href, ok := linkEl.Attr("href")
		if !ok || href == "" {
			return
		}

		title := strings.TrimSpace(linkEl.Text())

		companyNode := card.Find(selCompany).First()
		company := companyNode.AttrOr("data-company-name", "")
		if company == "" {
			company = strings.TrimSpace(companyNode.Text())
		}

		location := strings.TrimSpace(card.Find(selLocation).First().Text())

		jobs = append(jobs, dto.Job{
			Title:       title,
			Location:    location,
			URL:         href,
			CompanySlug: slugify(company),
		})
	})
	return jobs, nil
}

func ParseJobDetail(r io.Reader, url string) (dto.Job, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return dto.Job{}, fmt.Errorf("parse html: %w", err)
	}

	title := strings.TrimSpace(doc.Find(selTitle).First().Text())
	if title == "" {
		return dto.Job{}, fmt.Errorf("title not found (selector: %q)", selTitle)
	}

	companyNode := doc.Find(selCompany).First()
	company := companyNode.AttrOr("data-company-name", "")
	if company == "" {
		company = strings.TrimSpace(companyNode.Text())
	}

	location := strings.TrimSpace(doc.Find(selLocation).First().Text())

	updatedAt := time.Now().UTC()
	if dt, ok := doc.Find(selTime).First().Attr("datetime"); ok && dt != "" {
		if t, err := time.Parse(time.RFC3339, dt); err == nil {
			updatedAt = t
		} else if t, err := time.Parse("2006-01-02", dt); err == nil {
			updatedAt = t.UTC()
		}
	} else {
		slog.Warn("defaulted field",
			slog.String("source", "wis"),
			slog.String("field", "UpdatedAt"),
			slog.String("url", url),
		)
	}

	return dto.Job{
		Title:       title,
		Location:    location,
		URL:         url,
		CompanySlug: slugify(company),
		Source:      "wis",
		UpdatedAt:   updatedAt,
	}, nil
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case unicode.IsSpace(r) || r == '-':
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

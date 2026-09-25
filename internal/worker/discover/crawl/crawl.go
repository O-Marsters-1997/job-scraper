// Package crawl is the self-expanding step of company discovery: it visits
// the careers page of companies with a known domain but no resolved ATS
// board, and writes back whatever board it finds.
package crawl

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	neturl "net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	tickInterval = 6 * time.Hour
	batchSize    = 50

	fetchTimeout         = 15 * time.Second
	maxFetchesPerCompany = 5

	userAgent = userAgentToken + "/1.0 (+https://github.com/ollymarsters/job-scraper)"
)

var candidatePaths = []string{"/careers", "/jobs"}

const (
	selAnchorHref = "a[href]"
	selIframeSrc  = "iframe[src]"
	selScriptSrc  = "script[src]"
)

var careersNavPattern = regexp.MustCompile(`(?i)careers|jobs|join`)

// CompanyStore is the narrow subset of *db.DB the crawler needs.
type CompanyStore interface {
	ListCompaniesToCrawl(ctx context.Context, limit int) ([]dto.Company, error)
	TouchCompanyCrawled(ctx context.Context, id string) error
	UpsertCompany(ctx context.Context, c dto.CompanyUpsert) (dto.Company, error)
}

// Crawler resolves companies.domain rows with no known ats_source into an
// ATS board by fetching their careers page directly (never proxied) and
// scanning it for a recognisable board link or embed.
type Crawler struct {
	store  CompanyStore
	client *http.Client
}

func New(store CompanyStore) *Crawler {
	return &Crawler{
		store:  store,
		client: &http.Client{Timeout: fetchTimeout},
	}
}

// Run ticks every 6h, crawling a bounded batch of companies each time.
// Blocks until ctx is cancelled.
func (c *Crawler) Run(ctx context.Context) {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.tick(ctx)
		}
	}
}

func (c *Crawler) tick(ctx context.Context) {
	companies, err := c.store.ListCompaniesToCrawl(ctx, batchSize)
	if err != nil {
		slog.Error("crawl: list companies failed", slog.Any("err", err))
		return
	}
	slog.Info("crawl: batch fetched", slog.Int("count", len(companies)), slog.Int("limit", batchSize))

	for _, company := range companies {
		c.crawlCompany(ctx, company)
	}
}

func (c *Crawler) crawlCompany(ctx context.Context, company dto.Company) {
	log := slog.With(slog.String("company", company.Slug), slog.String("domain", company.Domain))

	if source, token, ok := c.resolve(ctx, log, company.Domain); ok {
		_, err := c.store.UpsertCompany(ctx, dto.CompanyUpsert{
			Slug:      company.Slug,
			Name:      company.Name,
			ATSSource: source,
			ATSToken:  token,
			Domain:    company.Domain,
		})
		if err != nil {
			log.Error("crawl: writeback failed", slog.Any("err", err))
		} else {
			log.Info("crawl: resolved ATS board", slog.String("source", source), slog.String("token", token))
		}
	}

	if err := c.store.TouchCompanyCrawled(ctx, company.ID); err != nil {
		log.Error("crawl: touch last_crawled_at failed", slog.Any("err", err))
	}
}

func (c *Crawler) resolve(ctx context.Context, log *slog.Logger, domain string) (source, token string, ok bool) {
	if domain == "" {
		return "", "", false
	}

	s := &session{client: c.client, host: domain, log: log}
	s.fetchRobots(ctx)

	for _, path := range candidatePaths {
		body, pageURL, fetched := s.get(ctx, path)
		if !fetched {
			continue
		}
		if source, token, ok = extract(body, pageURL); ok {
			return source, token, true
		}
	}

	homeBody, homeURL, fetched := s.get(ctx, "/")
	if !fetched {
		return "", "", false
	}
	if source, token, ok = extract(homeBody, homeURL); ok {
		return source, token, true
	}

	navURL, found := findCareersNavLink(bytes.NewReader(homeBody), homeURL)
	if !found {
		return "", "", false
	}
	navBody, navPageURL, fetched := s.getAbsolute(ctx, navURL)
	if !fetched {
		return "", "", false
	}
	return extract(navBody, navPageURL)
}

func extract(body []byte, pageURL *neturl.URL) (source, token string, ok bool) {
	links, err := ParseATSLinks(bytes.NewReader(body), pageURL)
	if err == nil {
		for _, link := range links {
			if source, token, ok = detect.ResolveBoard(link); ok {
				return source, token, true
			}
		}
	}
	return detect.SniffATS(body)
}

type session struct {
	client  *http.Client
	host    string
	polite  *politeness
	fetches int
	log     *slog.Logger
}

func (s *session) fetchRobots(ctx context.Context) {
	u := &neturl.URL{Scheme: "https", Host: s.host, Path: "/robots.txt"}
	body, err := s.fetchURL(ctx, u.String())
	s.fetches++
	if err != nil {
		s.log.Info("crawl: robots.txt unreachable, treating as allow-all", slog.Any("err", err))
		s.polite = allowAllPoliteness()
		return
	}
	polite, err := parseRobots(body)
	if err != nil {
		s.log.Warn("crawl: robots.txt parse failed, treating as allow-all", slog.Any("err", err))
		polite = allowAllPoliteness()
	}
	s.polite = polite
}

func (s *session) get(ctx context.Context, path string) ([]byte, *neturl.URL, bool) {
	return s.fetchIfAllowed(ctx, &neturl.URL{Scheme: "https", Host: s.host, Path: path})
}

func (s *session) getAbsolute(ctx context.Context, u *neturl.URL) ([]byte, *neturl.URL, bool) {
	return s.fetchIfAllowed(ctx, u)
}

func (s *session) fetchIfAllowed(ctx context.Context, u *neturl.URL) ([]byte, *neturl.URL, bool) {
	if s.fetches >= maxFetchesPerCompany {
		s.log.Info("crawl: fetch budget exhausted, skipping", slog.String("url", u.String()))
		return nil, nil, false
	}
	if !s.polite.allowed(u.Path) {
		s.log.Info("crawl: path disallowed by robots.txt", slog.String("path", u.Path))
		return nil, nil, false
	}
	if s.fetches > 0 {
		s.wait(ctx)
	}

	body, err := s.fetchURL(ctx, u.String())
	s.fetches++
	if err != nil {
		s.log.Warn("crawl: fetch failed", slog.String("url", u.String()), slog.Any("err", err))
		return nil, nil, false
	}
	return body, u, true
}

func (s *session) wait(ctx context.Context) {
	delay := defaultCrawlDelay
	if s.polite != nil {
		delay = s.polite.crawlDelay()
	}
	select {
	case <-ctx.Done():
	case <-time.After(delay):
	}
}

func (s *session) fetchURL(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// ParseATSLinks extracts candidate ATS URLs from a page: anchor hrefs, iframe
// srcs, and script srcs, resolved against base. It makes no ATS-detection
// decision itself — callers feed each URL through detect.ResolveBoard.
func ParseATSLinks(r io.Reader, base *neturl.URL) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("crawl: parsing HTML: %w", err)
	}

	var links []string
	collect := func(sel *goquery.Selection, attr string) {
		sel.Each(func(_ int, s *goquery.Selection) {
			val, ok := s.Attr(attr)
			if !ok || strings.TrimSpace(val) == "" {
				return
			}
			if resolved, err := resolveURL(base, val); err == nil {
				links = append(links, resolved)
			}
		})
	}

	collect(doc.Find(selAnchorHref), "href")
	collect(doc.Find(selIframeSrc), "src")
	collect(doc.Find(selScriptSrc), "src")

	return links, nil
}

func findCareersNavLink(r io.Reader, base *neturl.URL) (*neturl.URL, bool) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, false
	}

	var found *neturl.URL
	doc.Find(selAnchorHref).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		href, ok := s.Attr("href")
		if !ok || strings.TrimSpace(href) == "" {
			return true
		}
		text := strings.TrimSpace(s.Text())
		if !careersNavPattern.MatchString(text) && !careersNavPattern.MatchString(href) {
			return true
		}

		resolved, err := resolveURLValue(base, href)
		if err != nil || resolved.Host != base.Host {
			return true
		}
		found = resolved
		return false
	})

	return found, found != nil
}

func resolveURL(base *neturl.URL, ref string) (string, error) {
	resolved, err := resolveURLValue(base, ref)
	if err != nil {
		return "", err
	}
	return resolved.String(), nil
}

func resolveURLValue(base *neturl.URL, ref string) (*neturl.URL, error) {
	u, err := neturl.Parse(strings.TrimSpace(ref))
	if err != nil {
		return nil, err
	}
	return base.ResolveReference(u), nil
}

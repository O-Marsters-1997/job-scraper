package vcportfolio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/publicsuffix"

	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/slug"
	"github.com/ollymarsters/job-scraper/internal/worker/discover"
)

const (
	name      = "vcportfolio"
	userAgent = "Mozilla/5.0 (compatible; job-scraper/1.0)"
	interval  = 30 * 24 * time.Hour
)

func PortfolioURLs() []string {
	return []string{
		"https://seedcamp.com/our-companies/",
		"https://www.episode1.com/portfolio",
		"https://kindredcapital.vc/portfolio",
		"https://concept.vc/portfolio",
	}
}

func isNotCompany(domain string) bool {
	switch domain {
	case "linkedin.com", "twitter.com", "x.com", "facebook.com", "instagram.com", "youtube.com",
		"medium.com", "crunchbase.com", "google.com", "github.com", "typeform.com", "wikipedia.org":
		return true
	}
	return false
}

type Harvester struct {
	client *http.Client
	pages  []string
	get    discover.GetFunc
	log    *slog.Logger
}

func New(client *http.Client, pages []string, get discover.GetFunc, log *slog.Logger) *Harvester {
	return &Harvester{client: client, pages: pages, get: get, log: log}
}

func (h *Harvester) Name() string            { return name }
func (h *Harvester) Interval() time.Duration { return interval }

func (h *Harvester) Harvest(ctx context.Context) (discover.Harvest, error) {
	var out discover.Harvest
	seenBoards := map[discover.Board]bool{}
	var domains []string
	seenDomains := map[string]bool{}
	var errs []error
	for _, page := range h.pages {
		boards, pageDomains, err := h.fetchPage(ctx, page)
		if err != nil {
			if ctx.Err() != nil {
				return discover.Harvest{}, ctx.Err()
			}
			h.log.WarnContext(ctx, "vcportfolio: skipping page", slog.String("url", page), slog.Any(logger.KeyErr, err))
			errs = append(errs, err)
			continue
		}
		for _, b := range boards {
			if !seenBoards[b] {
				seenBoards[b] = true
				out.Boards = append(out.Boards, b)
			}
		}
		for _, d := range pageDomains {
			if !seenDomains[d] {
				seenDomains[d] = true
				domains = append(domains, d)
			}
		}
	}
	if len(h.pages) > 0 && len(errs) == len(h.pages) {
		return discover.Harvest{}, errors.Join(errs...)
	}

	for _, domain := range domains {
		company, ok := h.resolve(ctx, domain)
		if ctx.Err() != nil {
			return discover.Harvest{}, ctx.Err()
		}
		if !ok {
			out.Skipped++
			continue
		}
		out.Companies = append(out.Companies, company)
	}
	return out, nil
}

func (h *Harvester) fetchPage(ctx context.Context, pageURL string) ([]discover.Board, []string, error) {
	page, err := url.Parse(pageURL)
	if err != nil {
		return nil, nil, fmt.Errorf("parse page url: %w", err)
	}
	body, err := discover.Get(ctx, h.client, pageURL, userAgent)
	if err != nil {
		return nil, nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", pageURL, err)
	}
	own := site(page.Hostname())
	var boards []discover.Board
	var domains []string
	doc.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
		u, err := url.Parse(strings.TrimSpace(a.AttrOr("href", "")))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return
		}
		if source, token, ok := detect.ResolveBoard(u.String()); ok {
			boards = append(boards, discover.Board{Source: source, Token: token})
			return
		}
		domain := site(strings.ToLower(u.Hostname()))
		if domain == "" || domain == own || isNotCompany(domain) {
			return
		}
		domains = append(domains, domain)
	})
	return boards, domains, nil
}

func (h *Harvester) resolve(ctx context.Context, domain string) (discover.Company, bool) {
	label, _, _ := strings.Cut(domain, ".")
	companySlug := slug.Make(label)
	if companySlug == "" {
		return discover.Company{}, false
	}
	board, ok, err := discover.ResolveDomain(ctx, h.get, domain)
	if err != nil {
		h.log.WarnContext(ctx, "vcportfolio: could not resolve domain", slog.String(logger.KeyCompanySlug, companySlug), slog.Any(logger.KeyErr, err))
	}
	if !ok {
		return discover.Company{}, false
	}
	return discover.Company{Slug: companySlug, Name: slug.Humanize(companySlug), Domain: domain, Board: board}, true
}

func site(host string) string {
	if d, err := publicsuffix.EffectiveTLDPlusOne(host); err == nil {
		return d
	}
	return host
}

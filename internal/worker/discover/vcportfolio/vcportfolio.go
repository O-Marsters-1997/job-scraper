package vcportfolio

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
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

type Sources struct {
	Portfolios     []string
	GetroJobBoards []string
}

func DefaultSources() Sources {
	return Sources{
		Portfolios: []string{
			"https://seedcamp.com/our-companies/",
			"https://www.episode1.com/portfolio",
			"https://kindredcapital.vc/portfolio",
			"https://concept.vc/portfolio",
		},
		GetroJobBoards: []string{
			"https://talent.seedcamp.com",
			"https://careers.atomico.com",
			"https://jobs.dawncapital.com",
			"https://jobs.mmc.vc",
			"https://talent.octopusventures.com",
			"https://jobs.firstminute.capital",
			"https://jobs.kindredcapital.vc",
		},
	}
}

func isNotCompany(domain string) bool {
	switch domain {
	case "linkedin.com", "twitter.com", "x.com", "facebook.com", "instagram.com", "youtube.com",
		"medium.com", "crunchbase.com", "google.com", "github.com", "typeform.com", "wikipedia.org",
		"getro.com", "filepicker.io":
		return true
	}
	return false
}

type Harvester struct {
	client  *http.Client
	sources Sources
	get     discover.GetFunc
	log     *slog.Logger
}

func New(client *http.Client, sources Sources, get discover.GetFunc, log *slog.Logger) *Harvester {
	return &Harvester{client: client, sources: sources, get: get, log: log}
}

func (h *Harvester) Name() string            { return name }
func (h *Harvester) Interval() time.Duration { return interval }

func (h *Harvester) Harvest(ctx context.Context) (discover.Harvest, error) {
	var out discover.Harvest
	seenBoards := map[discover.Board]bool{}
	var domains []string
	seenDomains := map[string]bool{}
	var errs []error
	pages := slices.Clone(h.sources.Portfolios)
	for _, board := range h.sources.GetroJobBoards {
		companyPages, err := h.boardCompanyPages(ctx, board)
		if err != nil {
			if ctx.Err() != nil {
				return discover.Harvest{}, ctx.Err()
			}
			h.log.WarnContext(ctx, "vcportfolio: skipping job board", slog.String("url", board), slog.Any(logger.KeyErr, err))
			errs = append(errs, err)
			continue
		}
		pages = append(pages, companyPages...)
	}
	for _, page := range pages {
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
	if len(errs) > 0 && len(domains) == 0 && len(out.Boards) == 0 {
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

func (h *Harvester) boardCompanyPages(ctx context.Context, board string) ([]string, error) {
	body, err := discover.Get(ctx, h.client, board+"/sitemap.xml", userAgent)
	if err != nil {
		return nil, err
	}
	var sitemap struct {
		Locs []string `xml:"url>loc"`
	}
	if err := xml.Unmarshal(body, &sitemap); err != nil {
		return nil, fmt.Errorf("parse %s sitemap: %w", board, err)
	}
	var pages []string
	seen := map[string]bool{}
	for _, loc := range sitemap.Locs {
		u, err := url.Parse(loc)
		if err != nil {
			continue
		}
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) < 2 || parts[0] != "companies" || seen[parts[1]] {
			continue
		}
		seen[parts[1]] = true
		pages = append(pages, board+"/companies/"+parts[1])
	}
	return pages, nil
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

package wellfound

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/slug"
	"github.com/ollymarsters/job-scraper/internal/worker/discover"
	source "github.com/ollymarsters/job-scraper/internal/worker/sources/wellfound"
)

const (
	name     = "wellfound"
	interval = 7 * 24 * time.Hour
)

var LocationPages = []string{"london", "united-kingdom", "manchester", "edinburgh"}

type VerifiedSlugs interface {
	ListVerifiedCompanySlugs(ctx context.Context, slugs []string) ([]string, error)
}

type Harvester struct {
	pages     discover.GetFunc
	sites     discover.GetFunc
	verified  VerifiedSlugs
	locations []string
}

var _ discover.Harvester = (*Harvester)(nil)

func New(pages, sites discover.GetFunc, verified VerifiedSlugs, locations []string) *Harvester {
	return &Harvester{pages: pages, sites: sites, verified: verified, locations: locations}
}

func (h *Harvester) Name() string            { return name }
func (h *Harvester) Interval() time.Duration { return interval }

type candidate struct {
	slug, name, listingURL string
}

func (h *Harvester) Harvest(ctx context.Context) (discover.Harvest, error) {
	candidates, err := h.candidates(ctx)
	if err != nil {
		return discover.Harvest{}, err
	}
	slugs := make([]string, len(candidates))
	for i, c := range candidates {
		slugs[i] = c.slug
	}
	known, err := h.verified.ListVerifiedCompanySlugs(ctx, slugs)
	if err != nil {
		return discover.Harvest{}, fmt.Errorf("list verified companies: %w", err)
	}
	verified := make(map[string]bool, len(known))
	for _, s := range known {
		verified[s] = true
	}

	var out discover.Harvest
	for _, c := range candidates {
		if verified[c.slug] {
			continue
		}
		company, ok := h.resolve(ctx, c)
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

func (h *Harvester) candidates(ctx context.Context) ([]candidate, error) {
	var out []candidate
	seen := make(map[string]bool)
	var errs []error
	for _, location := range h.locations {
		listings, err := h.locationListings(ctx, location)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if err != nil {
			slog.WarnContext(ctx, "wellfound: could not read location", slog.String("location", location), slog.Any(logger.KeyErr, err))
			errs = append(errs, err)
		}
		for _, l := range listings {
			companySlug := slug.Company(l.Company)
			if companySlug == "" || seen[companySlug] {
				continue
			}
			seen[companySlug] = true
			out = append(out, candidate{slug: companySlug, name: l.Company, listingURL: l.URL})
		}
	}
	if len(errs) > 0 && len(errs) == len(h.locations) {
		return nil, fmt.Errorf("read wellfound locations: %w", errors.Join(errs...))
	}
	return out, nil
}

func (h *Harvester) locationListings(ctx context.Context, location string) ([]source.ATSListing, error) {
	var out []source.ATSListing
	for page, pageCount := 1, 1; page <= pageCount; page++ {
		pageURL, err := source.LocationPageURL(location, page)
		if err != nil {
			return out, err
		}
		body, err := h.pages(ctx, pageURL)
		if err != nil {
			return out, err
		}
		var listings []source.ATSListing
		listings, pageCount, err = source.ParseATSListings(bytes.NewReader(body))
		if err != nil {
			return out, fmt.Errorf("parse %s: %w", pageURL, err)
		}
		out = append(out, listings...)
	}
	return out, nil
}

func (h *Harvester) resolve(ctx context.Context, c candidate) (discover.Company, bool) {
	body, err := h.pages(ctx, c.listingURL)
	if err != nil {
		slog.WarnContext(ctx, "wellfound: could not fetch listing", slog.String(logger.KeyCompanySlug, c.slug), slog.Any(logger.KeyErr, err))
		return discover.Company{}, false
	}
	domain := hostOf(websiteOf(body))
	if domain == "" {
		return discover.Company{}, false
	}
	board, ok, err := discover.ResolveDomain(ctx, h.sites, domain)
	if err != nil {
		slog.WarnContext(ctx, "wellfound: could not resolve domain", slog.String(logger.KeyCompanySlug, c.slug), slog.Any(logger.KeyErr, err))
	}
	if !ok {
		return discover.Company{}, false
	}
	return discover.Company{Slug: c.slug, Name: c.name, Domain: domain, Board: board}, true
}

func websiteOf(detailPage []byte) string {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(detailPage))
	if err != nil {
		return ""
	}
	var site string
	doc.Find(`script[type="application/ld+json"]`).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		var posting struct {
			HiringOrganization struct {
				SameAs string `json:"sameAs"`
			} `json:"hiringOrganization"`
		}
		if json.Unmarshal([]byte(s.Text()), &posting) == nil {
			site = posting.HiringOrganization.SameAs
		}
		return site == ""
	})
	return site
}

func hostOf(website string) string {
	u, err := url.Parse(website)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(u.Hostname(), "www.")
}

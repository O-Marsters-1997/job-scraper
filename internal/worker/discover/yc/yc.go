package yc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/slug"
	"github.com/ollymarsters/job-scraper/internal/worker/discover"
)

const (
	CompaniesURL = "https://yc-oss.github.io/api/companies/all.json"

	name      = "yc"
	userAgent = "Mozilla/5.0 (compatible; job-scraper/1.0)"
	interval  = 7 * 24 * time.Hour
)

type Harvester struct {
	client       *http.Client
	companiesURL string
	get          discover.GetFunc
}

func New(client *http.Client, companiesURL string, get discover.GetFunc) *Harvester {
	return &Harvester{client: client, companiesURL: companiesURL, get: get}
}

func (h *Harvester) Name() string            { return name }
func (h *Harvester) Interval() time.Duration { return interval }

type company struct {
	Name         string   `json:"name"`
	Slug         string   `json:"slug"`
	Website      string   `json:"website"`
	Status       string   `json:"status"`
	IsHiring     bool     `json:"isHiring"`
	Regions      []string `json:"regions"`
	AllLocations string   `json:"all_locations"`
}

func (c company) isUKHiring() bool {
	if c.Status != "Active" || !c.IsHiring {
		return false
	}
	return slices.Contains(c.Regions, "United Kingdom") ||
		strings.Contains(c.AllLocations, "London") || strings.Contains(c.AllLocations, "United Kingdom")
}

func (h *Harvester) Harvest(ctx context.Context) (discover.Harvest, error) {
	body, err := discover.Get(ctx, h.client, h.companiesURL, userAgent)
	if err != nil {
		return discover.Harvest{}, err
	}
	var all []company
	if err := json.Unmarshal(body, &all); err != nil {
		return discover.Harvest{}, fmt.Errorf("parse yc companies: %w", err)
	}

	var out discover.Harvest
	seen := make(map[string]bool)
	for _, c := range all {
		if !c.isUKHiring() {
			continue
		}
		companySlug := slug.Make(c.Slug)
		domain := hostOf(c.Website)
		if companySlug == "" || domain == "" || seen[companySlug] {
			out.Skipped++
			continue
		}
		seen[companySlug] = true
		board, ok, err := discover.ResolveDomain(ctx, h.get, domain)
		if ctx.Err() != nil {
			return discover.Harvest{}, ctx.Err()
		}
		if err != nil {
			slog.WarnContext(ctx, "yc: could not resolve domain", slog.String(logger.KeyCompanySlug, companySlug), slog.Any(logger.KeyErr, err))
		}
		if !ok {
			out.Skipped++
			continue
		}
		out.Companies = append(out.Companies, discover.Company{Slug: companySlug, Name: c.Name, Domain: domain, Board: board})
	}
	return out, nil
}

func hostOf(website string) string {
	u, err := url.Parse(website)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(u.Hostname(), "www.")
}

// Package yc harvests Y Combinator's public company directory
// (https://api.ycombinator.com/v0.1/companies) into discover.Company records.
package yc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/worker/discover"
)

const (
	baseURL  = "https://api.ycombinator.com/v0.1/companies?page=1"
	maxPages = 100
)

type Harvester struct {
	client  *http.Client
	baseURL string
}

func New() *Harvester {
	return &Harvester{client: &http.Client{Timeout: 15 * time.Second, Transport: logger.FetchTransport(nil)}, baseURL: baseURL}
}

func (h *Harvester) WithBaseURL(u string) *Harvester {
	h.baseURL = u
	return h
}

func (h *Harvester) Name() string { return "yc" }

func (h *Harvester) Harvest(ctx context.Context) ([]discover.Company, error) {
	var all []discover.Company
	next := h.baseURL
	for page := 0; next != ""; page++ {
		if page >= maxPages {
			slog.WarnContext(ctx, "yc harvester: page cap reached, stopping early", slog.Int("cap", maxPages))
			break
		}
		body, err := h.fetch(ctx, next)
		if err != nil {
			return nil, err
		}
		companies, nextPage, err := parseCompanies(body)
		if err != nil {
			return nil, err
		}
		all = append(all, companies...)
		next = nextPage
	}
	return all, nil
}

func (h *Harvester) fetch(ctx context.Context, pageURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", pageURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: status %d", pageURL, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

type apiResponse struct {
	Companies []apiCompany `json:"companies"`
	NextPage  string       `json:"nextPage"`
}

type apiCompany struct {
	Name    string `json:"name"`
	Website string `json:"website"`
}

func parseCompanies(body []byte) (companies []discover.Company, next string, err error) {
	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, "", fmt.Errorf("unmarshal yc companies: %w", err)
	}

	companies = make([]discover.Company, 0, len(resp.Companies))
	for _, c := range resp.Companies {
		if c.Name == "" && c.Website == "" {
			continue
		}
		companies = append(companies, discover.Company{
			Name:   c.Name,
			Domain: hostOf(c.Website),
		})
	}
	return companies, resp.NextPage, nil
}

func hostOf(website string) string {
	if website == "" {
		return ""
	}
	raw := website
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(u.Host), "www.")
}

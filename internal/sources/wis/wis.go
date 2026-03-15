package wis

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	startURL       = "https://workinstartups.com/search?loc=86384&pp=50&sb=date&sd=down&q=product%20engineer&per_page=50"
	defaultTimeout = 15 * time.Second
)

// Scraper implements sources.Source for workinstartups.com.
type Scraper struct {
	client *http.Client
}

// New constructs a WIS Scraper.
func New() *Scraper {
	return &Scraper{
		client: &http.Client{Timeout: defaultTimeout},
	}
}

// Name implements sources.Source.
func (s *Scraper) Name() string { return "wis" }

// FetchJobs implements sources.Source. It fetches the WIS search page and
// parses job listings from the HTML response.
func (s *Scraper) FetchJobs(ctx context.Context) ([]dto.Job, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, startURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; job-scraper/1.0)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}

	jobs, err := ParseHTML(resp.Body)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	for i := range jobs {
		jobs[i].UpdatedAt = now
	}

	slog.Debug("wis page parsed", slog.String("source", "wis"), slog.Int("count", len(jobs)))
	return jobs, nil
}

// ParseHTML parses a WIS search results page and returns job listings.
// UpdatedAt is left as zero — callers that need a timestamp should set it themselves.
func ParseHTML(r io.Reader) ([]dto.Job, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}
	return extractJobs(doc), nil
}

// extractJobs walks the HTML tree collecting job cards.
// Each job is wrapped in a <div data-aid="..."> element.
func extractJobs(n *html.Node) []dto.Job {
	var jobs []dto.Job
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			if attr(n, "data-aid") != "" {
				if j, ok := parseJobCard(n); ok {
					jobs = append(jobs, j)
					return // don't recurse into the card
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return jobs
}

func parseJobCard(card *html.Node) (dto.Job, bool) {
	var title, jobURL, company, location string

	// h2 > a holds the title and job detail URL
	h2 := findFirst(card, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "h2"
	})
	if h2 != nil {
		a := findFirst(h2, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "a"
		})
		if a != nil {
			title = strings.TrimSpace(textContent(a))
			jobURL = attr(a, "href")
		}
	}

	if title == "" || jobURL == "" {
		return dto.Job{}, false
	}

	// <div class="ui-company" data-company-name="...">
	companyDiv := findFirst(card, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "ui-company")
	})
	if companyDiv != nil {
		company = attr(companyDiv, "data-company-name")
		if company == "" {
			company = strings.TrimSpace(textContent(companyDiv))
		}
	}

	// <div class="ui-location ...">
	locationDiv := findFirst(card, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "ui-location")
	})
	if locationDiv != nil {
		location = strings.TrimSpace(textContent(locationDiv))
	}

	return dto.Job{
		Title:       title,
		URL:         jobURL,
		CompanySlug: company,
		Location:    location,
		Source:      "wis",
	}, true
}

func findFirst(n *html.Node, pred func(*html.Node) bool) *html.Node {
	if pred(n) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findFirst(c, pred); found != nil {
			return found
		}
	}
	return nil
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hasClass(n *html.Node, class string) bool {
	return slices.Contains(strings.Fields(attr(n, "class")), class)
}

func textContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(textContent(c))
	}
	return sb.String()
}

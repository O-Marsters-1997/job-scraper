package wis

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	startURL       = "https://workinstartups.com/search?loc=86384&pp=50&sb=date&sd=down&q=product%20engineer&per_page=50"
	defaultTimeout = 15 * time.Second
	resultsPerPage = 50
	minWait        = 2 * time.Second
	maxWait        = 7 * time.Second
)

type Scraper struct {
	client *http.Client
}

func New() *Scraper {
	return &Scraper{
		client: &http.Client{Timeout: defaultTimeout},
	}
}

func (s *Scraper) Name() string { return "wis" }

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

func pageURL(page int) string {
	if page <= 1 {
		return startURL
	}
	return fmt.Sprintf("%s&p=%d", startURL, page)
}

// fetchPage fetches a single page and returns the jobs and, for page 1, the
// total result count. totalCount is 0 for pages other than 1.
func (s *Scraper) fetchPage(ctx context.Context, page int) (jobs []dto.Job, totalCount int, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL(page), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; job-scraper/1.0)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("read body: %w", err)
	}

	jobs, err = ParseHTML(bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}

	if page == 1 {
		totalCount, err = ParseTotalCount(bytes.NewReader(body))
		if err != nil {
			return nil, 0, err
		}
	}

	return jobs, totalCount, nil
}

func (s *Scraper) Iterate(ctx context.Context) ([]dto.Job, error) {
	page1, total, err := s.fetchPage(ctx, 1)
	if err != nil {
		return nil, fmt.Errorf("wis page 1: %w", err)
	}

	pages := TotalPages(total)
	slog.Info("wis iterating", slog.String("source", "wis"), slog.Int("total", total), slog.Int("pages", pages))

	all := make([]dto.Job, 0, total)
	all = append(all, page1...)

	for p := 2; p <= pages; p++ {
		wait := minWait + time.Duration(rand.Int64N(int64(maxWait-minWait)))
		select {
		case <-ctx.Done():
			return all, ctx.Err()
		case <-time.After(wait):
		}

		slog.Debug("wis fetching page", slog.String("source", "wis"), slog.Int("page", p), slog.Int("of", pages))

		pageJobs, _, err := s.fetchPage(ctx, p)
		if err != nil {
			slog.Error("wis page failed", slog.String("source", "wis"), slog.Int("page", p), slog.Any("err", err))
			continue
		}
		all = append(all, pageJobs...)
	}

	now := time.Now()
	for i := range all {
		all[i].UpdatedAt = now
	}

	slog.Info("wis done", slog.String("source", "wis"), slog.Int("count", len(all)))
	return all, nil
}

func ParseTotalCount(r io.Reader) (int, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return 0, fmt.Errorf("parse html: %w", err)
	}
	span := findFirst(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "span" && attr(n, "data-cy-count") != ""
	})
	if span == nil {
		return 0, fmt.Errorf("data-cy-count element not found")
	}
	count, err := strconv.Atoi(attr(span, "data-cy-count"))
	if err != nil {
		return 0, fmt.Errorf("parse data-cy-count: %w", err)
	}
	return count, nil
}

func TotalPages(totalCount int) int {
	return (totalCount + resultsPerPage - 1) / resultsPerPage
}

func ParseHTML(r io.Reader) ([]dto.Job, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}
	return extractJobs(doc), nil
}

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

	companyDiv := findFirst(card, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "ui-company")
	})
	if companyDiv != nil {
		company = attr(companyDiv, "data-company-name")
		if company == "" {
			company = strings.TrimSpace(textContent(companyDiv))
		}
	}

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

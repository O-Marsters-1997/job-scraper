package wis

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"golang.org/x/net/html"
)

const (
	startURL        = "https://workinstartups.com/search?loc=86384&pp=50&sb=date&sd=down&q=product%20engineer&per_page=50"
	defaultTimeout  = 15 * time.Second
	resultsPerPage  = 50
	minWait         = 2 * time.Second
	maxWait         = 7 * time.Second
	defaultSchedule = "0 */6 * * *" // every 6 hours
)

type Scraper struct {
	client *http.Client
}

func New() *Scraper {
	return &Scraper{
		client: &http.Client{Timeout: defaultTimeout},
	}
}

func (s *Scraper) Name() string                     { return "wis" }
func (s *Scraper) FetchSchedule() string            { return defaultSchedule }
func (s *Scraper) MinScrapeInterval() time.Duration { return 5 * time.Hour }

func (s *Scraper) FetchURLs(ctx context.Context) ([]string, error) {
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

	urls, err := ParseURLs(resp.Body)
	if err != nil {
		return nil, err
	}

	slog.Debug("wis page parsed", slog.String("source", "wis"), slog.Int("count", len(urls)))
	return urls, nil
}

func pageURL(page int) string {
	if page <= 1 {
		return startURL
	}
	return fmt.Sprintf("%s&p=%d", startURL, page)
}

// fetchPage fetches a single page and returns the URLs and, for page 1, the
// total result count. totalCount is 0 for pages other than 1.
func (s *Scraper) fetchPage(ctx context.Context, page int) (urls []string, totalCount int, err error) {
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

	urls, err = ParseURLs(bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}

	if page == 1 {
		totalCount, err = ParseTotalCount(bytes.NewReader(body))
		if err != nil {
			return nil, 0, err
		}
	}

	return urls, totalCount, nil
}

func (s *Scraper) Iterate(ctx context.Context, fn func(context.Context, []string) (bool, error)) error {
	page1, total, err := s.fetchPage(ctx, 1)
	if err != nil {
		return fmt.Errorf("wis page 1: %w", err)
	}

	pages := TotalPages(total)
	slog.Info("wis iterating", slog.String("source", "wis"), slog.Int("total", total), slog.Int("pages", pages))

	stop, err := fn(ctx, page1)
	if err != nil || stop {
		return err
	}

	for p := 2; p <= pages; p++ {
		wait := minWait + time.Duration(rand.Int64N(int64(maxWait-minWait)))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}

		slog.Debug("wis fetching page", slog.String("source", "wis"), slog.Int("page", p), slog.Int("of", pages))

		pageURLs, _, err := s.fetchPage(ctx, p)
		if err != nil {
			slog.Error("wis page failed", slog.String("source", "wis"), slog.Int("page", p), slog.Any("err", err))
			continue
		}

		stop, err = fn(ctx, pageURLs)
		if err != nil {
			return err
		}
		if stop {
			slog.Info("wis early stop", slog.String("source", "wis"), slog.Int("page", p))
			break
		}
	}

	return nil
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

func ParseURLs(r io.Reader) ([]string, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}
	return extractURLs(doc), nil
}

func extractURLs(n *html.Node) []string {
	var urls []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			if attr(n, "data-aid") != "" {
				if u := parseJobCardURL(n); u != "" {
					urls = append(urls, u)
					return // don't recurse into the card
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return urls
}

func parseJobCardURL(card *html.Node) string {
	h2 := findFirst(card, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "h2"
	})
	if h2 == nil {
		return ""
	}
	a := findFirst(h2, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "a"
	})
	if a == nil {
		return ""
	}
	return attr(a, "href")
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

func (s *Scraper) CanHandle(url string) bool {
	return strings.Contains(url, "workinstartups.com")
}

func (s *Scraper) GetDetails(ctx context.Context, url string) (dto.Job, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return dto.Job{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; job-scraper/1.0)")

	resp, err := s.client.Do(req)
	if err != nil {
		return dto.Job{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return dto.Job{}, fmt.Errorf("unexpected status %s", resp.Status)
	}

	return ParseJobDetail(resp.Body, url)
}

func ParseJobDetail(r io.Reader, url string) (dto.Job, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return dto.Job{}, fmt.Errorf("parse html: %w", err)
	}

	title := strings.TrimSpace(textContent(findFirst(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "h1"
	})))

	companyName := ""
	if companyNode := findFirst(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && hasClass(n, "ui-company")
	}); companyNode != nil {
		if v := attr(companyNode, "data-company-name"); v != "" {
			companyName = v
		} else {
			companyName = strings.TrimSpace(textContent(companyNode))
		}
	}

	location := ""
	if locNode := findFirst(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && hasClass(n, "ui-location")
	}); locNode != nil {
		location = strings.TrimSpace(textContent(locNode))
	}

	updatedAt := time.Now().UTC()
	if timeNode := findFirst(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "time" && attr(n, "datetime") != ""
	}); timeNode != nil {
		if t, err := time.Parse(time.RFC3339, attr(timeNode, "datetime")); err == nil {
			updatedAt = t
		} else if t, err := time.Parse("2006-01-02", attr(timeNode, "datetime")); err == nil {
			updatedAt = t.UTC()
		}
	}

	return dto.Job{
		Title:       title,
		Location:    location,
		URL:         url,
		CompanySlug: slugify(companyName),
		Source:      "wis",
		UpdatedAt:   updatedAt,
	}, nil
}

func hasClass(n *html.Node, class string) bool {
	for c := range strings.FieldsSeq(attr(n, "class")) {
		if c == class {
			return true
		}
	}
	return false
}

func textContent(n *html.Node) string {
	if n == nil {
		return ""
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
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

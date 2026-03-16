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
	"time"

	"golang.org/x/net/html"

	"github.com/ollymarsters/job-scraper/internal/sources"
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

func (s *Scraper) Iterate(ctx context.Context, filter sources.URLFilter) ([]string, error) {
	page1, total, err := s.fetchPage(ctx, 1)
	if err != nil {
		return nil, fmt.Errorf("wis page 1: %w", err)
	}

	pages := TotalPages(total)
	slog.Info("wis iterating", slog.String("source", "wis"), slog.Int("total", total), slog.Int("pages", pages))

	newURLs, err := filter(ctx, page1)
	if err != nil {
		slog.Error("wis filter failed", slog.String("source", "wis"), slog.Int("page", 1), slog.Any("err", err))
		newURLs = page1
	}

	all := make([]string, 0, total)
	all = append(all, newURLs...)

	if len(newURLs) == 0 {
		slog.Info("wis early stop", slog.String("source", "wis"), slog.Int("page", 1))
		return all, nil
	}

	for p := 2; p <= pages; p++ {
		wait := minWait + time.Duration(rand.Int64N(int64(maxWait-minWait)))
		select {
		case <-ctx.Done():
			return all, ctx.Err()
		case <-time.After(wait):
		}

		slog.Debug("wis fetching page", slog.String("source", "wis"), slog.Int("page", p), slog.Int("of", pages))

		pageURLs, _, err := s.fetchPage(ctx, p)
		if err != nil {
			slog.Error("wis page failed", slog.String("source", "wis"), slog.Int("page", p), slog.Any("err", err))
			continue
		}

		newURLs, err = filter(ctx, pageURLs)
		if err != nil {
			slog.Error("wis filter failed", slog.String("source", "wis"), slog.Int("page", p), slog.Any("err", err))
			newURLs = pageURLs
		}

		all = append(all, newURLs...)

		if len(newURLs) == 0 {
			slog.Info("wis early stop", slog.String("source", "wis"), slog.Int("page", p))
			break
		}
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

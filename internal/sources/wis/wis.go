package wis

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/net/html"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

const (
	startURL       = "https://workinstartups.com/search?q=product+engineer&w=uk&per_page=50"
	resultsPerPage = 50
)

type Scraper struct{ sources.PaginatedBase }

var _ sources.Source = (*Scraper)(nil)

func New() *Scraper {
	return &Scraper{sources.NewBase(sources.Config{
		Name:              "wis",
		URLPrefix:         "https://workinstartups.com",
		Schedule:          "0 */6 * * *",
		MinScrapeInterval: 5 * time.Hour,
	})}
}

func pageURL(page int) string {
	if page <= 1 {
		return startURL
	}
	return fmt.Sprintf("%s&p=%d", startURL, page)
}

func (s *Scraper) fetchPage(ctx context.Context, page int) (urls []string, totalCount int, err error) {
	body, err := s.Get(ctx, pageURL(page))
	if err != nil {
		return nil, 0, err
	}

	jobs, err := ParseURLs(bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	urls = make([]string, len(jobs))
	for i, j := range jobs {
		urls[i] = j.URL
	}

	if page == 1 {
		totalCount, err = ParseTotalCount(bytes.NewReader(body))
		if err != nil {
			return nil, 0, err
		}
	}

	return urls, totalCount, nil
}

func (s *Scraper) ParseURLs(r io.Reader) ([]dto.Job, error) {
	return ParseURLs(r)
}

func (s *Scraper) ParseJobDetail(r io.Reader, url string) (dto.Job, error) {
	return ParseJobDetail(r, url)
}

func (s *Scraper) Iterate(ctx context.Context, fn func(context.Context, []string) (bool, error)) error {
	return s.IteratePages(ctx, fn, s.fetchPage, resultsPerPage)
}

func (s *Scraper) GetDetails(ctx context.Context, url string) (dto.Job, error) {
	body, err := s.Get(ctx, url)
	if err != nil {
		return dto.Job{}, err
	}
	return ParseJobDetail(bytes.NewReader(body), url)
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

func ParseURLs(r io.Reader) ([]dto.Job, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}
	return extractURLs(doc), nil
}

func extractURLs(n *html.Node) []dto.Job {
	var jobs []dto.Job
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			if attr(n, "data-aid") != "" {
				if u := parseJobCardURL(n); u != "" {
					jobs = append(jobs, dto.Job{URL: u})
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

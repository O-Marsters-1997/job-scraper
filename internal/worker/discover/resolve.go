package discover

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/ollymarsters/job-scraper/internal/detect"
)

var careersLink = regexp.MustCompile(`(?i)careers|jobs|join[- ]us|vacancies|work[- ]with[- ]us|opportunities`)

var seedPaths = []string{"/careers", "/jobs", "/join-us", "/work-with-us", "/vacancies"}

type GetFunc func(ctx context.Context, url string) ([]byte, error)

func ResolveDomain(ctx context.Context, get GetFunc, domain string) (Board, bool, error) {
	home := "https://" + domain
	doc, body, err := fetchDoc(ctx, get, home)
	if err != nil {
		return Board{}, false, err
	}
	if b, ok := findBoard(doc, body); ok {
		return b, true, nil
	}

	hop := careersHop(doc, home)
	if hop == "" {
		b, ok := probeSeedPaths(ctx, get, home)
		return b, ok, nil
	}
	doc, body, err = fetchDoc(ctx, get, hop)
	if err != nil {
		return Board{}, false, err
	}
	b, ok := findBoard(doc, body)
	return b, ok, nil
}

func probeSeedPaths(ctx context.Context, get GetFunc, home string) (Board, bool) {
	for _, path := range seedPaths {
		if ctx.Err() != nil {
			break
		}
		doc, body, err := fetchDoc(ctx, get, home+path)
		if err != nil {
			continue
		}
		if b, ok := findBoard(doc, body); ok {
			return b, true
		}
	}
	return Board{}, false
}

func fetchDoc(ctx context.Context, get GetFunc, pageURL string) (*goquery.Document, []byte, error) {
	body, err := get(ctx, pageURL)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch %s: %w", pageURL, err)
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", pageURL, err)
	}
	return doc, body, nil
}

func findBoard(doc *goquery.Document, body []byte) (Board, bool) {
	var found Board
	var ok bool
	doc.Find("a[href], iframe[src]").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		raw, _ := s.Attr("href")
		if raw == "" {
			raw, _ = s.Attr("src")
		}
		source, token, resolved := detect.ResolveBoard(raw)
		found, ok = Board{Source: source, Token: token}, resolved
		return !resolved
	})
	if ok {
		return found, true
	}
	source, token, sniffed := detect.SniffATS(body)
	return Board{Source: source, Token: token}, sniffed
}

func careersHop(doc *goquery.Document, home string) string {
	base, err := url.Parse(home)
	if err != nil {
		return ""
	}
	pathHop, textHop := "", ""
	doc.Find("a[href]").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		raw, _ := s.Attr("href")
		ref, err := url.Parse(raw)
		if err != nil {
			return true
		}
		abs := base.ResolveReference(ref)
		if abs.Scheme != "https" && abs.Scheme != "http" {
			return true
		}
		switch {
		case careersLink.MatchString(abs.Path):
			pathHop = abs.String()
			return false
		case textHop == "" && careersLink.MatchString(strings.TrimSpace(s.Text())):
			textHop = abs.String()
		}
		return true
	})
	if pathHop != "" {
		return pathHop
	}
	return textHop
}

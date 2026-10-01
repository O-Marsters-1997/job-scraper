// Package wttj harvests Welcome to the Jungle companies from its sitemap.
package wttj

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/slug"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
	"github.com/ollymarsters/job-scraper/internal/worker/discover"
)

const (
	// SitemapURL lists every live WTTJ job and company.
	SitemapURL = "https://app.welcometothejungle.com/sitemap1.xml"

	source      = "wttj"
	userAgent   = "Mozilla/5.0 (compatible; job-scraper/1.0)"
	interval    = 24 * time.Hour
	companyPath = "/companies/"
)

type Harvester struct {
	client     *http.Client
	sitemapURL string
}

func New(client *http.Client, sitemapURL string) *Harvester {
	return &Harvester{client: client, sitemapURL: sitemapURL}
}

func (h *Harvester) Name() string            { return source }
func (h *Harvester) Interval() time.Duration { return interval }

func (h *Harvester) Harvest(ctx context.Context) (discover.Harvest, error) {
	body, err := discover.Get(ctx, h.client, h.sitemapURL, userAgent)
	if err != nil {
		return discover.Harvest{}, err
	}
	tokens, skipped, err := parseSitemap(body)
	if err != nil {
		return discover.Harvest{}, err
	}
	out := discover.Harvest{Skipped: skipped}
	seen := make(map[string]bool, len(tokens))
	for _, token := range tokens {
		companySlug := slug.Make(token)
		if companySlug == "" || seen[companySlug] {
			out.Skipped++
			continue
		}
		seen[companySlug] = true
		out.Companies = append(out.Companies, discover.Company{
			Slug: companySlug, Name: slug.Humanize(companySlug), Board: discover.Board{Source: source, Token: token},
		})
	}
	return out, nil
}

func parseSitemap(body []byte) (tokens []string, skipped int, err error) {
	dec := xml.NewDecoder(bytes.NewReader(body))
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("parse sitemap: %w", err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "loc" {
			continue
		}
		var loc string
		if err := dec.DecodeElement(&loc, &start); err != nil {
			return nil, 0, fmt.Errorf("parse sitemap: %w", err)
		}
		_, name, found := strings.Cut(loc, companyPath)
		switch {
		case !found:
		case sourcespec.ValidBoardToken(name):
			tokens = append(tokens, name)
		default:
			skipped++
		}
	}
	return tokens, skipped, nil
}

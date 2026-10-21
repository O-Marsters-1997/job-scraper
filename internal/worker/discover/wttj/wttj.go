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

	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/slug"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
	"github.com/ollymarsters/job-scraper/internal/worker/discover"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/wttj"
)

const (
	SitemapURL = "https://app.welcometothejungle.com/sitemap1.xml"

	source    = "wttj"
	userAgent = "Mozilla/5.0 (compatible; job-scraper/1.0)"
	interval  = 24 * time.Hour
	jobPath   = "/jobs/"
)

type Harvester struct {
	client     *http.Client
	sitemapURL string
	api        *wttj.Client
}

func New(client *http.Client, sitemapURL string, api *wttj.Client) *Harvester {
	return &Harvester{client: client, sitemapURL: sitemapURL, api: api}
}

func (h *Harvester) Name() string            { return source }
func (h *Harvester) Interval() time.Duration { return interval }

func (h *Harvester) Harvest(ctx context.Context) (discover.Harvest, error) {
	body, err := discover.Get(ctx, h.client, h.sitemapURL, userAgent)
	if err != nil {
		return discover.Harvest{}, err
	}
	ids, skipped, err := parseSitemap(body)
	if err != nil {
		return discover.Harvest{}, err
	}
	listings, err := h.api.EligibleListings(ctx, ids)
	if err != nil && (!errors.Is(err, sources.ErrDeferred) || len(listings) == 0 || ctx.Err() != nil) {
		return discover.Harvest{}, err
	}
	out := discover.Harvest{Skipped: skipped}
	seenBoards := map[discover.Board]bool{}
	seenCompanies := map[string]bool{}
	for _, l := range listings {
		if board, ok := atsBoard(l.ApplyURL); ok {
			if !seenBoards[board] {
				seenBoards[board] = true
				out.Boards = append(out.Boards, board)
			}
			continue
		}
		companySlug := slug.Make(l.CompanyToken)
		if companySlug == "" || !sourcespec.ValidBoardToken(l.CompanyToken) {
			out.Skipped++
			continue
		}
		if !seenCompanies[companySlug] {
			seenCompanies[companySlug] = true
			out.Companies = append(out.Companies, discover.Company{
				Slug: companySlug, Name: l.CompanyName, Board: discover.Board{Source: source, Token: l.CompanyToken},
			})
		}
	}
	return out, nil
}

func atsBoard(applyURL string) (discover.Board, bool) {
	src, token, ok := detect.ResolveBoard(applyURL)
	if !ok || !sourcespec.ValidBoardToken(token) {
		return discover.Board{}, false
	}
	if role, known := sourcespec.SourceRole(src); !known || role != sourcespec.RoleATS {
		return discover.Board{}, false
	}
	return discover.Board{Source: src, Token: token}, true
}

func parseSitemap(body []byte) (ids []string, skipped int, err error) {
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
		_, id, found := strings.Cut(loc, jobPath)
		switch {
		case !found:
		case sourcespec.ValidBoardToken(id):
			ids = append(ids, id)
		default:
			skipped++
		}
	}
	return ids, skipped, nil
}

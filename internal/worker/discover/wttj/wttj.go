package wttj

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
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
	interval  = 30 * time.Minute
	jobPath   = "/jobs/"

	outcomeBackfill   = "backfill"
	outcomeIneligible = "ineligible"
	outcomeExported   = "exported"
)

type SeenStore interface {
	HarvestSeenIDs(ctx context.Context, source string) ([]string, error)
	RecordHarvestSeenIDs(ctx context.Context, source, outcome string, ids []string) error
	RetireHarvestSeenIDs(ctx context.Context, source string, ids []string) error
}

type Exporter interface {
	Export(ctx context.Context, job dto.Job) error
}

type Harvester struct {
	client     *http.Client
	sitemapURL string
	api        *wttj.Client
	seen       SeenStore
	exporter   Exporter
}

func New(client *http.Client, sitemapURL string, api *wttj.Client, seen SeenStore, exporter Exporter) *Harvester {
	return &Harvester{client: client, sitemapURL: sitemapURL, api: api, seen: seen, exporter: exporter}
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
	known, err := h.seen.HarvestSeenIDs(ctx, source)
	if err != nil {
		return discover.Harvest{}, err
	}
	out := discover.Harvest{Skipped: skipped}
	if len(known) == 0 {
		return out, h.seen.RecordHarvestSeenIDs(ctx, source, outcomeBackfill, ids)
	}
	inSitemap := make(map[string]bool, len(ids))
	for _, id := range ids {
		inSitemap[id] = true
	}
	knownSet := make(map[string]bool, len(known))
	for _, id := range known {
		knownSet[id] = true
	}
	var fresh []string
	for id := range inSitemap {
		if !knownSet[id] {
			fresh = append(fresh, id)
		}
	}
	slices.Sort(fresh)
	if err := h.handleNew(ctx, fresh, &out); err != nil {
		return discover.Harvest{}, err
	}
	if len(ids) > 0 {
		gone := slices.DeleteFunc(known, func(id string) bool { return inSitemap[id] })
		if err := h.seen.RetireHarvestSeenIDs(ctx, source, gone); err != nil {
			return discover.Harvest{}, err
		}
	}
	return out, nil
}

func (h *Harvester) handleNew(ctx context.Context, fresh []string, out *discover.Harvest) error {
	listings, err := h.api.EligibleListings(ctx, fresh)
	if err != nil && (!errors.Is(err, sources.ErrDeferred) || len(listings) == 0 || ctx.Err() != nil) {
		return err
	}
	seenBoards := map[discover.Board]bool{}
	seenCompanies := map[string]bool{}
	var native []string
	var ineligible []string
	if err == nil {
		eligible := make(map[string]bool, len(listings))
		for _, l := range listings {
			eligible[l.ID] = true
		}
		for _, id := range fresh {
			if !eligible[id] {
				ineligible = append(ineligible, id)
			}
		}
	}
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
			ineligible = append(ineligible, l.ID)
			continue
		}
		native = append(native, l.ID)
		if !seenCompanies[companySlug] {
			seenCompanies[companySlug] = true
			out.Companies = append(out.Companies, discover.Company{
				Slug: companySlug, Name: l.CompanyName, Board: discover.Board{Source: source, Token: l.CompanyToken},
			})
		}
	}
	if err := h.seen.RecordHarvestSeenIDs(ctx, source, outcomeIneligible, ineligible); err != nil {
		return err
	}
	return h.export(ctx, native)
}

func (h *Harvester) export(ctx context.Context, ids []string) error {
	jobs, err := h.api.Jobs(ctx, ids)
	var exported []string
	for _, job := range jobs {
		if expErr := h.exporter.Export(ctx, job); expErr != nil {
			slog.WarnContext(ctx, "wttj harvest: could not export job", slog.String(logger.KeyURL, job.URL), slog.Any(logger.KeyErr, expErr))
			continue
		}
		exported = append(exported, job.ProviderPostingID)
	}
	if recErr := h.seen.RecordHarvestSeenIDs(ctx, source, outcomeExported, exported); recErr != nil {
		return recErr
	}
	if err != nil {
		slog.WarnContext(ctx, "wttj harvest: could not fetch job details", slog.Any(logger.KeyErr, err))
	}
	return nil
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

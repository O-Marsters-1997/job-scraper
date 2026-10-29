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
	interval  = 10 * time.Minute
	jobPath   = "/jobs/"

	sitemapInterval = 24 * time.Hour
	absentFor       = 7 * 24 * time.Hour

	listingBase = "https://app.welcometothejungle.com" + jobPath

	outcomeBackfill   = "backfill"
	outcomeIneligible = "ineligible"
	outcomeExported   = "exported"
	outcomeATSCheck   = "ats_check"
)

type SeenStore interface {
	HarvestSeenIDs(ctx context.Context, source string) ([]string, error)
	RecordHarvestSeenIDs(ctx context.Context, source, outcome string, ids []string) error
	MarkHarvestIDsInSitemap(ctx context.Context, source string, ids []string, seenAt time.Time) error
	CloseJobsAbsentFromSitemap(ctx context.Context, source string, absentSince time.Time) error
	HarvestCompanyCounts(ctx context.Context, source string) (map[string]int, error)
	RecordHarvestCompanyCounts(ctx context.Context, source string, counts map[string]int) error
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
	now        func() time.Time
	sitemapAt  time.Time
}

func New(client *http.Client, sitemapURL string, api *wttj.Client, seen SeenStore, exporter Exporter, now func() time.Time) *Harvester {
	return &Harvester{client: client, sitemapURL: sitemapURL, api: api, seen: seen, exporter: exporter, now: now}
}

func (h *Harvester) Name() string            { return source }
func (h *Harvester) Interval() time.Duration { return interval }

func (h *Harvester) Harvest(ctx context.Context) (discover.Harvest, error) {
	known, err := h.seen.HarvestSeenIDs(ctx, source)
	if err != nil {
		return discover.Harvest{}, err
	}
	now := h.now()
	var out discover.Harvest
	var sitemapIDs []string
	sitemapDue := now.Sub(h.sitemapAt) >= sitemapInterval
	if sitemapDue {
		if sitemapIDs, out.Skipped, err = h.fetchSitemap(ctx); err != nil {
			return discover.Harvest{}, err
		}
		if len(known) == 0 {
			h.sitemapAt = now
			return out, h.seen.RecordHarvestSeenIDs(ctx, source, outcomeBackfill, sitemapIDs)
		}
		if err := h.markSitemap(ctx, sitemapIDs, now); err != nil {
			return discover.Harvest{}, err
		}
	}
	counts, liveIDs, err := h.liveFeed(ctx)
	if err != nil {
		return discover.Harvest{}, err
	}
	complete, err := h.handleNew(ctx, now, unseen(known, sitemapIDs, liveIDs), &out)
	if err != nil {
		return discover.Harvest{}, err
	}
	if !complete {
		return out, nil
	}
	if sitemapDue {
		h.sitemapAt = now
	}
	return out, h.seen.RecordHarvestCompanyCounts(ctx, source, counts)
}

func (h *Harvester) fetchSitemap(ctx context.Context) ([]string, int, error) {
	body, err := discover.Get(ctx, h.client, h.sitemapURL, userAgent)
	if err != nil {
		return nil, 0, err
	}
	return parseSitemap(body)
}

func (h *Harvester) markSitemap(ctx context.Context, ids []string, now time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	if err := h.seen.MarkHarvestIDsInSitemap(ctx, source, ids, now); err != nil {
		return err
	}
	return h.seen.CloseJobsAbsentFromSitemap(ctx, source, now.Add(-absentFor))
}

func (h *Harvester) liveFeed(ctx context.Context) (map[string]int, []string, error) {
	prev, err := h.seen.HarvestCompanyCounts(ctx, source)
	if err != nil {
		return nil, nil, err
	}
	counts, err := h.api.LiveJobCounts(ctx)
	if err != nil {
		return nil, nil, err
	}
	if len(prev) == 0 {
		return counts, nil, nil
	}
	var risen []string
	for token, n := range counts {
		if n > prev[token] && sourcespec.ValidBoardToken(token) {
			risen = append(risen, token)
		}
	}
	slices.Sort(risen)
	ids, err := h.api.LiveJobIDs(ctx, risen)
	if err != nil {
		return nil, nil, err
	}
	return counts, ids, nil
}

func unseen(known []string, candidates ...[]string) []string {
	skip := make(map[string]bool, len(known))
	for _, id := range known {
		skip[id] = true
	}
	var fresh []string
	for _, ids := range candidates {
		for _, id := range ids {
			if !skip[id] {
				skip[id] = true
				fresh = append(fresh, id)
			}
		}
	}
	slices.Sort(fresh)
	return fresh
}

func (h *Harvester) handleNew(ctx context.Context, now time.Time, fresh []string, out *discover.Harvest) (complete bool, err error) {
	listings, err := h.api.EligibleListings(ctx, fresh)
	if err != nil && (!errors.Is(err, sources.ErrDeferred) || len(listings) == 0 || ctx.Err() != nil) {
		return false, err
	}
	complete = err == nil
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
		if !l.Current(now) {
			ineligible = append(ineligible, l.ID)
			continue
		}
		if board, ok := atsBoard(l.ApplyURL); ok {
			board.ListingURL, board.ApplyURL = listingBase+l.ID, l.ApplyURL
			out.Checks = append(out.Checks, discover.Check{ID: l.ID, Board: board})
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
		return false, err
	}
	return complete, h.export(ctx, native)
}

func (h *Harvester) RecordChecks(ctx context.Context, ids []string) error {
	return h.seen.RecordHarvestSeenIDs(ctx, source, outcomeATSCheck, ids)
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

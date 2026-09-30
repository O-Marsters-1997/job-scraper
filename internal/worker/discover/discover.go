// Package discover drains code-registered Harvesters into the shared companies
// catalog on a fixed cadence, deduplicating candidates and gating each
// harvester to one run per interval.
package discover

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/slug"
)

// Company is a harvested catalog candidate. Zero-value fields are unknown.
type Company struct {
	Name, Domain, ATSSource, ATSToken, LinkedInCompanyID string
}

// Get fetches url with client and returns the body of a 200 response. A non-empty
// userAgent is sent as the User-Agent header.
func Get(ctx context.Context, client *http.Client, url, userAgent string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: status %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// Harvester yields company catalog candidates from one external source.
type Harvester interface {
	// Name identifies the harvester in logs and as its gate key ("harvest:" + Name()).
	Name() string
	Harvest(ctx context.Context) ([]Company, error)
}

// ScrapeGate persists each harvester's successful run time.
type ScrapeGate interface {
	SetLastScraped(ctx context.Context, source string) error
	GetLastScraped(ctx context.Context, source string) (time.Time, bool, error)
}

// CompanyUpserter is the narrow subset of jobsearch's Boards() the harvest
// loop needs to write catalog candidates.
type CompanyUpserter interface {
	UpsertCompany(ctx context.Context, c dto.CompanyUpsert) (dto.Company, error)
}

const (
	harvestInterval = 24 * time.Hour
	gateKeyPrefix   = "harvest:"
)

// Runner drains harvesters into the companies catalog on a fixed cadence.
type Runner struct {
	harvesters []Harvester
	companies  CompanyUpserter
	gate       ScrapeGate
}

func NewRunner(hs []Harvester, companies CompanyUpserter, gate ScrapeGate) *Runner {
	return &Runner{harvesters: hs, companies: companies, gate: gate}
}

// RunOnce harvests every due harvester and joins their failures.
func (r *Runner) RunOnce(ctx context.Context) error {
	var errs []error
	for _, h := range r.harvesters {
		if err := r.runIfDue(ctx, h); err != nil {
			errs = append(errs, fmt.Errorf("harvest %s: %w", h.Name(), err))
		}
	}
	return errors.Join(errs...)
}

func (r *Runner) runIfDue(ctx context.Context, h Harvester) error {
	key := gateKeyPrefix + h.Name()
	log := slog.With(slog.String("harvester", h.Name()))

	last, ok, err := r.gate.GetLastScraped(ctx, key)
	if err != nil {
		log.WarnContext(ctx, "could not read last harvested, proceeding", slog.Any(logger.KeyErr, err))
	}
	if ok && time.Since(last) < harvestInterval {
		log.InfoContext(ctx, "skipping harvest: ran recently", slog.Duration("ago", time.Since(last)))
		return nil
	}

	log.InfoContext(ctx, "harvest: running")
	companies, err := h.Harvest(ctx)
	if err != nil {
		return err
	}

	upserted := 0
	for _, c := range companies {
		if err := r.upsert(ctx, c); err != nil {
			log.ErrorContext(ctx, "upsert failed", slog.String("name", c.Name), slog.Any(logger.KeyErr, err))
			continue
		}
		upserted++
	}
	log.InfoContext(ctx, "harvest: completed", slog.Int(logger.KeyCount, len(companies)), slog.Int("upserted", upserted))

	if err := r.gate.SetLastScraped(ctx, key); err != nil {
		log.ErrorContext(ctx, "could not set last harvested", slog.Any(logger.KeyErr, err))
	}
	return nil
}

func (r *Runner) upsert(ctx context.Context, c Company) error {
	slugSource := c.Name
	if slugSource == "" {
		slugSource = c.Domain
	}
	if slugSource == "" {
		return nil
	}

	_, err := r.companies.UpsertCompany(ctx, dto.CompanyUpsert{
		Slug:              slug.Make(slugSource),
		Name:              c.Name,
		ATSSource:         c.ATSSource,
		ATSToken:          c.ATSToken,
		Domain:            c.Domain,
		LinkedInCompanyID: c.LinkedInCompanyID,
	})
	return err
}

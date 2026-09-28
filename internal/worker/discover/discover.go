// Package discover drains code-registered Harvesters into the shared companies
// catalog on a fixed cadence, deduplicating candidates and gating each
// harvester to one run per interval.
package discover

import (
	"context"
	"log/slog"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/slug"
)

// Company is a harvested catalog candidate. Zero-value fields are unknown.
type Company struct {
	Name, Domain, ATSSource, ATSToken, LinkedInCompanyID string
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
	tickInterval    = time.Hour
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

// Run ticks hourly, checking each harvester's 24h gate before harvesting.
// Blocks until ctx is cancelled.
func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.tick(ctx)
		}
	}
}

func (r *Runner) tick(ctx context.Context) {
	for _, h := range r.harvesters {
		r.runIfDue(ctx, h)
	}
}

func (r *Runner) runIfDue(ctx context.Context, h Harvester) {
	key := gateKeyPrefix + h.Name()
	log := slog.With(slog.String("harvester", h.Name()))

	last, ok, err := r.gate.GetLastScraped(ctx, key)
	if err != nil {
		log.WarnContext(ctx, "could not read last harvested, proceeding", slog.Any(logger.KeyErr, err))
	}
	if ok && time.Since(last) < harvestInterval {
		log.InfoContext(ctx, "skipping harvest: ran recently", slog.Duration("ago", time.Since(last)))
		return
	}

	companies, err := h.Harvest(ctx)
	if err != nil {
		log.ErrorContext(ctx, "harvest failed", slog.Any(logger.KeyErr, err))
		return
	}

	for _, c := range companies {
		if err := r.upsert(ctx, c); err != nil {
			log.ErrorContext(ctx, "upsert failed", slog.String("name", c.Name), slog.Any(logger.KeyErr, err))
		}
	}

	if err := r.gate.SetLastScraped(ctx, key); err != nil {
		log.ErrorContext(ctx, "could not set last harvested", slog.Any(logger.KeyErr, err))
	}
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

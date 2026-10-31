// Package discover turns code-registered Harvesters into board_discover tasks
// on a per-harvester cadence, gating each harvester to one run per interval.
package discover

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

// Board is an ATS board a harvester found, identified by source and token.
type Board struct {
	Source, Token string
	ListingURL    string
	ApplyURL      string
}

// Company is a company a harvester found, with the candidate Board it would be polled through.
type Company struct {
	Slug, Name, Domain string
	Board              Board
}

type Check struct {
	ID    string
	Board Board
}

type CheckRecorder interface {
	RecordChecks(ctx context.Context, ids []string) error
}

// Harvest is what one harvester run found. Skipped counts candidates that
// resolved to no board or an unsupported source. Recheck publishes Boards that
// are already verified; otherwise they are dropped.
type Harvest struct {
	Boards    []Board
	Checks    []Check
	Companies []Company
	Skipped   int
	Recheck   bool
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
		return nil, fmt.Errorf("fetch %s: %w", url, &sources.StatusError{Code: resp.StatusCode, Status: resp.Status})
	}
	return io.ReadAll(resp.Body)
}

// Harvester yields ATS boards from one external source.
type Harvester interface {
	// Name identifies the harvester in logs and as its gate key ("harvest:" + Name()).
	Name() string
	// Interval is the minimum time between successful runs.
	Interval() time.Duration
	Harvest(ctx context.Context) (Harvest, error)
}

// ScrapeGate persists each harvester's successful run time.
type ScrapeGate interface {
	SetLastScraped(ctx context.Context, source string) error
	GetLastScraped(ctx context.Context, source string) (time.Time, bool, error)
}

// Publisher is the subset of the queue broker the Runner needs.
type Publisher interface {
	Publish(ctx context.Context, task queue.Task) error
}

// Catalog is the Company and Board store a harvested Company is written to.
type Catalog interface {
	UpsertCompany(ctx context.Context, c dto.CompanyUpsert) (dto.Company, error)
	ListVerifiedCompanySlugs(ctx context.Context, slugs []string) ([]string, error)
	GetVerifiedBoardID(ctx context.Context, source, token string) (string, error)
	DiscoverMissedSince(ctx context.Context, source, token string, since time.Time) (bool, error)
	UpsertCandidateBoard(ctx context.Context, companyID, source, token string) (dto.CompanyBoard, error)
}

const (
	gateKeyPrefix = "harvest:"
	missWindow    = 30 * 24 * time.Hour
)

// Runner publishes a board_discover task per harvested board, and records each
// harvested Company that has no verified Board, with a candidate Board and a
// board_discover task for it.
type Runner struct {
	harvesters []Harvester
	publisher  Publisher
	gate       ScrapeGate
	catalog    Catalog
}

func NewRunner(hs []Harvester, publisher Publisher, gate ScrapeGate, catalog Catalog) *Runner {
	return &Runner{harvesters: hs, publisher: publisher, gate: gate, catalog: catalog}
}

// RegisterMetrics exports each harvester's interval, labelled to match jobscraper_harvest_age_seconds.
func (r *Runner) RegisterMetrics(reg prometheus.Registerer) {
	interval := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "jobscraper_harvest_interval_seconds",
		Help: "How often each catalog harvester is due to run.",
	}, []string{"harvester"})
	for _, h := range r.harvesters {
		interval.WithLabelValues(gateKeyPrefix + h.Name()).Set(h.Interval().Seconds())
	}
	reg.MustRegister(interval)
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
	if ok && time.Since(last) < h.Interval() {
		log.InfoContext(ctx, "skipping harvest: ran recently", slog.Duration("ago", time.Since(last)))
		return nil
	}

	log.InfoContext(ctx, "harvest: running")
	found, err := h.Harvest(ctx)
	if err != nil {
		return err
	}

	verified, err := r.verifiedSlugs(ctx, found.Companies)
	if err != nil {
		return fmt.Errorf("list verified companies: %w", err)
	}

	published := 0
	for _, b := range found.Boards {
		if !found.Recheck {
			_, err := r.catalog.GetVerifiedBoardID(ctx, b.Source, b.Token)
			if err == nil {
				found.Skipped++
				continue
			}
			if !errors.Is(err, data.ErrNotFound) {
				log.WarnContext(ctx, "could not check board verified, publishing", slog.String(logger.KeySource, b.Source), slog.String("token", b.Token), slog.Any(logger.KeyErr, err))
			}
			if r.discoverMissed(ctx, log, b) {
				found.Skipped++
				continue
			}
		}
		if err := r.publisher.Publish(ctx, discoverTask(b, h.Name())); err != nil {
			log.ErrorContext(ctx, "publish failed", slog.String(logger.KeySource, b.Source), slog.String("token", b.Token), slog.Any(logger.KeyErr, err))
			continue
		}
		published++
	}
	published += r.publishChecks(ctx, log, h, found.Checks)
	recorded := r.recordUndiscovered(ctx, log, h.Name(), found.Companies, verified)
	log.InfoContext(ctx, "harvest: completed", slog.String("event", "harvest.run"),
		slog.Int("candidates", len(found.Boards)+len(found.Checks)+len(found.Companies)), slog.Int("resolved", published+recorded),
		slog.Int("skipped", found.Skipped))

	if err := r.gate.SetLastScraped(ctx, key); err != nil {
		log.ErrorContext(ctx, "could not set last harvested", slog.Any(logger.KeyErr, err))
	}
	return nil
}

func discoverTask(b Board, via string) queue.Task {
	return queue.Task{Version: 1, ID: uuid.NewString(), Source: b.Source, Kind: queue.BoardDiscoverTask, BoardToken: b.Token, Via: via, ListingURL: b.ListingURL, ApplyURL: b.ApplyURL, RequireUK: true}
}

func (r *Runner) discoverMissed(ctx context.Context, log *slog.Logger, b Board) bool {
	missed, err := r.catalog.DiscoverMissedSince(ctx, b.Source, b.Token, time.Now().Add(-missWindow))
	if err != nil {
		log.WarnContext(ctx, "could not check board missed, publishing", slog.String(logger.KeySource, b.Source), slog.String("token", b.Token), slog.Any(logger.KeyErr, err))
	}
	return missed
}

func (r *Runner) publishChecks(ctx context.Context, log *slog.Logger, h Harvester, checks []Check) int {
	via := h.Name()
	published := 0
	var handled []string
	for _, c := range checks {
		b := c.Board
		task := discoverTask(b, via)
		boardID, err := r.catalog.GetVerifiedBoardID(ctx, b.Source, b.Token)
		switch {
		case err == nil:
			task = queue.Task{Version: 1, ID: uuid.NewString(), Source: b.Source, Kind: queue.BoardCheckTask, BoardID: boardID, Via: via, ListingURL: b.ListingURL, ApplyURL: b.ApplyURL, Urgent: true, Manual: true}
		case !errors.Is(err, data.ErrNotFound):
			log.WarnContext(ctx, "could not check board verified, publishing", slog.String(logger.KeySource, b.Source), slog.String("token", b.Token), slog.Any(logger.KeyErr, err))
		case r.discoverMissed(ctx, log, b):
			handled = append(handled, c.ID)
			continue
		}
		if err := r.publisher.Publish(ctx, task); err != nil {
			log.ErrorContext(ctx, "publish failed", slog.String(logger.KeySource, b.Source), slog.String("token", b.Token), slog.Any(logger.KeyErr, err))
			continue
		}
		handled = append(handled, c.ID)
		published++
	}
	if rec, ok := h.(CheckRecorder); ok && len(handled) > 0 {
		if err := rec.RecordChecks(ctx, handled); err != nil {
			log.ErrorContext(ctx, "could not record checks", slog.Any(logger.KeyErr, err))
		}
	}
	return published
}

func (r *Runner) verifiedSlugs(ctx context.Context, companies []Company) (map[string]bool, error) {
	slugs := make([]string, len(companies))
	for i, c := range companies {
		slugs[i] = c.Slug
	}
	found, err := r.catalog.ListVerifiedCompanySlugs(ctx, slugs)
	if err != nil {
		return nil, err
	}
	verified := make(map[string]bool, len(found))
	for _, s := range found {
		verified[s] = true
	}
	return verified, nil
}

func (r *Runner) recordUndiscovered(ctx context.Context, log *slog.Logger, via string, companies []Company, verified map[string]bool) int {
	recorded := 0
	for _, c := range companies {
		if verified[c.Slug] {
			continue
		}
		if err := r.recordCompany(ctx, via, c); err != nil {
			log.WarnContext(ctx, "harvest: could not record company", slog.String(logger.KeyCompanySlug, c.Slug), slog.Any(logger.KeyErr, err))
			continue
		}
		recorded++
	}
	return recorded
}

func (r *Runner) recordCompany(ctx context.Context, via string, c Company) error {
	company, err := r.catalog.UpsertCompany(ctx, dto.CompanyUpsert{Slug: c.Slug, Name: c.Name, Domain: c.Domain, Kind: dto.KindDirect})
	if err != nil {
		return err
	}
	board, err := r.catalog.UpsertCandidateBoard(ctx, company.ID, c.Board.Source, c.Board.Token)
	if err != nil {
		return err
	}
	if board.Status != dto.BoardCandidate {
		return nil
	}
	return r.publisher.Publish(ctx, queue.Task{
		Version: 1, ID: uuid.NewString(), Source: board.Source, Kind: queue.BoardDiscoverTask,
		CompanyID: company.ID, BoardToken: board.BoardToken, Via: via, RequireUK: true,
	})
}

package scraper

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/ollymarsters/job-scraper/internal/candidates"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/score"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

type Orchestrator struct {
	srcs         []sources.Source
	db           providers.JobProvider
	q            queue.JobQueue
	cfgDB        providers.SearchConfigProvider // nil means no reject filter
	exporter     JobExporter                    // nil means no ATS egress
	cr           *cron.Cron
	wg           sync.WaitGroup
	buildSources func(ctx context.Context) ([]sources.Source, error) // nil → use static srcs
	buildTarget  func(target dto.SourceTarget) []sources.Source      // nil → no on-demand scrape
	force        bool                                                // bypass MinScrapeInterval gate when true
	candidates   *candidates.Service
}

func New(srcs []sources.Source, db providers.JobProvider, q queue.JobQueue) *Orchestrator {
	return &Orchestrator{srcs: srcs, db: db, q: q}
}

// WithSourceReloader wires in functions to reload sources from DB at each tick
// and to build a one-off source for on-demand scraping.
// Calling this is optional; without it the static srcs slice is used.
func (o *Orchestrator) WithSourceReloader(
	buildAll func(ctx context.Context) ([]sources.Source, error),
	buildOne func(target dto.SourceTarget) []sources.Source,
) *Orchestrator {
	o.buildSources = buildAll
	o.buildTarget = buildOne
	return o
}

// WithRejectFilter wires in the search-config store so both ingestion paths
// drop jobs that every user's exclusion filters reject.
func (o *Orchestrator) WithRejectFilter(cfgDB providers.SearchConfigProvider) {
	o.cfgDB = cfgDB
}

func (o *Orchestrator) WithCandidates(store candidates.Store) *Orchestrator {
	o.candidates = candidates.New(store, o.q)
	return o
}

// WithForceScrape bypasses the per-source MinScrapeInterval gate on every tick.
// Use locally to test the full scrape→enqueue→ingest flow without wiping Valkey state.
func (o *Orchestrator) WithForceScrape(force bool) *Orchestrator {
	o.force = force
	return o
}

// WithExporter wires in the egress port used by the ATS path.
func (o *Orchestrator) WithExporter(exp JobExporter) *Orchestrator {
	o.exporter = exp
	return o
}

func (o *Orchestrator) Start(ctx context.Context) error {
	// When force is set, run the initial tick synchronously so the queue is
	// pre-populated before the caller starts the queue consumer.
	if o.force {
		o.tick(ctx)
	} else {
		o.wg.Add(1)
		go func() {
			defer o.wg.Done()
			o.tick(ctx)
		}()
	}

	o.cr = cron.New()

	// If no reloader is set, tick falls back to the static srcs slice.
	if _, err := o.cr.AddFunc(sources.DefaultSchedule, func() {
		slog.Info("cron: starting scrape tick")
		// wg.Add must be called before the goroutine starts; cron calls this
		// func synchronously so it's safe here.
		o.wg.Add(1)
		go func() {
			defer o.wg.Done()
			o.tick(ctx)
		}()
	}); err != nil {
		o.cr.Stop()
		return fmt.Errorf("schedule scrape tick (%s): %w", sources.DefaultSchedule, err)
	}
	slog.Info("cron: scheduled scrape tick", slog.String("schedule", sources.DefaultSchedule))

	o.cr.Start()
	return nil
}

// tick reloads the source list from DB (if a reloader is set) and fans out
// runIfReady for every source. New targets join rotation automatically on the
// next tick without a worker restart.
func (o *Orchestrator) tick(ctx context.Context) {
	srcs := o.srcs
	if o.buildSources != nil {
		rebuilt, err := o.buildSources(ctx)
		if err != nil {
			slog.Error("reload sources failed", slog.Any("err", err))
		} else {
			srcs = rebuilt
		}
	}

	var wg sync.WaitGroup
	for _, src := range srcs {
		wg.Add(1)
		go func(s sources.Source) {
			defer wg.Done()
			o.runIfReady(ctx, s)
		}(src)
	}
	wg.Wait()
}

// ScrapeTarget builds a one-off source for the given target and runs a scrape
// immediately. It bypasses the MinScrapeInterval gate (user-requested) and does
// NOT write scrape:last, so the regular scheduled rotation is unaffected.
func (o *Orchestrator) ScrapeTarget(ctx context.Context, target dto.SourceTarget) error {
	if o.buildTarget == nil {
		return fmt.Errorf("scraper: no buildTarget func registered")
	}
	srcs := o.buildTarget(target)
	for _, src := range srcs {
		role, _ := sources.SourceRole(target.Source)
		if role == sources.RoleDiscovery && o.candidates != nil {
			if err := o.runDiscovery(ctx, src, target); err != nil {
				return err
			}
			continue
		}
		if err := o.run(ctx, src); err != nil {
			return err
		}
	}
	return nil
}

func (o *Orchestrator) runDiscovery(ctx context.Context, src sources.Source, target dto.SourceTarget) error {
	config := dto.SearchConfig{UserID: target.UserID}
	if o.cfgDB != nil {
		stored, err := o.cfgDB.GetSearchConfig(ctx, target.UserID)
		if err != nil && err != providers.ErrNotFound {
			return err
		}
		if err == nil {
			config = stored
		}
	}
	return src.Iterate(ctx, func(ctx context.Context, cards []dto.Job) (bool, error) {
		for i := range cards {
			if detect.Detect(cards[i].URL) == detect.Aggregator {
				if rewritten, _, ok := detect.RewriteToATS(cards[i].URL); ok {
					cards[i].URL = rewritten
				}
			}
		}
		return false, o.candidates.CapturePage(ctx, target, cards, config)
	})
}

func (o *Orchestrator) Stop() {
	if o.cr != nil {
		o.cr.Stop()
	}
	o.wg.Wait()
}

func (o *Orchestrator) runIfReady(ctx context.Context, src sources.Source) {
	cfg := src.Cfg()
	log := slog.With(slog.String("source", cfg.Name))

	// ATS sources are gated per-target in SQL (ListDueSourceTargets) via
	// check_interval_minutes; the platform-wide scrape:last gate below only
	// applies to discovery sources, which have no per-target freshness column.
	if role, _ := sources.SourceRole(cfg.Name); role == sources.RoleATS {
		if err := o.run(ctx, src); err != nil {
			log.Error("scrape failed", slog.Any("err", err))
		}
		return
	}

	last, ok, err := o.q.GetLastScraped(ctx, cfg.Name)
	if err != nil {
		log.Warn("could not read last scraped, proceeding",
			slog.Any("err", err),
		)
	}
	if !o.force && ok && time.Since(last) < cfg.MinScrapeInterval {
		log.Info("skipping scrape: ran recently", slog.Duration("ago", time.Since(last)))
		return
	}
	if err := o.run(ctx, src); err != nil {
		log.Error("scrape failed", slog.Any("err", err))
		return
	}
	if err := o.q.SetLastScraped(ctx, cfg.Name); err != nil {
		log.Error("could not set last scraped", slog.Any("err", err))
	}
}

type userGate struct {
	userID string
	cfg    dto.SearchConfig
}

func (g userGate) rejects(job dto.Job) (string, bool) {
	return score.Reject(job, g.cfg)
}

func (o *Orchestrator) loadGates(ctx context.Context, source string) []userGate {
	if o.cfgDB == nil {
		return nil
	}
	cfgs, err := o.cfgDB.ListSearchConfigs(ctx)
	if err != nil {
		slog.Warn("could not load search configs, skipping reject filter",
			slog.String("source", source),
			slog.Any("err", err),
		)
		return nil
	}
	gates := make([]userGate, len(cfgs))
	for i, cfg := range cfgs {
		gates[i] = userGate{userID: cfg.UserID, cfg: cfg}
	}
	return gates
}

// keepJob reports whether job survives every user's reject filter, i.e. at
// least one user's filter does not reject it. When every user rejects it,
// each user's reason is logged so filters can be tuned.
func keepJob(job dto.Job, gates []userGate) bool {
	if len(gates) == 0 {
		return true
	}
	reasons := make([]string, 0, len(gates))
	for _, g := range gates {
		reason, rejected := g.rejects(job)
		if !rejected {
			return true
		}
		reasons = append(reasons, g.userID+": "+reason)
	}
	slog.Info("job rejected by all user filters",
		slog.String("title", job.Title),
		slog.String("url", job.URL),
		slog.Any("reasons", reasons),
	)
	return false
}

type atsPath struct {
	exporter JobExporter
	name     string
	gates    []userGate
}

func (p *atsPath) onPage(ctx context.Context, jobs []dto.Job) (bool, error) {
	log := slog.With(slog.String("source", p.name))

	passing := make([]dto.Job, 0, len(jobs))
	for _, j := range jobs {
		if j.URL == "" {
			continue
		}
		if keepJob(j, p.gates) {
			passing = append(passing, j)
		}
	}

	if len(passing) == 0 {
		log.Info("ats jobs fetched, none passed relevance gate", slog.Int("fetched", len(jobs)))
		return false, nil
	}

	if err := p.exporter.BulkExport(ctx, passing); err != nil {
		return false, err
	}

	log.Info("ats jobs published", slog.Int("count", len(passing)))
	return false, nil
}

type htmlPath struct {
	db    providers.JobProvider
	q     queue.JobQueue
	gates []userGate
	seen  map[string]struct{}
}

func (p *htmlPath) onPage(ctx context.Context, jobs []dto.Job) (bool, error) {
	type partial struct {
		url string
		job dto.Job
	}
	candidates := make([]partial, 0, len(jobs))
	for _, j := range jobs {
		u := j.URL
		if u == "" {
			continue
		}
		if detect.Detect(u) == detect.Aggregator {
			if rewritten, _, ok := detect.RewriteToATS(u); ok {
				u = rewritten
			}
		}
		candidates = append(candidates, partial{url: u, job: j})
	}

	urls := make([]string, len(candidates))
	for i, c := range candidates {
		urls[i] = c.url
	}

	newURLs, err := p.db.NewURLs(ctx, urls)
	if err != nil {
		slog.Warn("filter failed, using all URLs", slog.Any("err", err))
		newURLs = urls
	}

	newSet := make(map[string]struct{}, len(newURLs))
	for _, u := range newURLs {
		newSet[u] = struct{}{}
	}

	if skipped := len(candidates) - len(newURLs); skipped > 0 {
		slog.Info("skipping already scraped URLs",
			slog.Int("total_urls", len(candidates)),
			slog.Int("skipped", skipped),
		)
	}

	queued := make([]dto.QueuedJob, 0, len(candidates))
	freshCount, gateFiltered := 0, 0
	for _, c := range candidates {
		if _, isNew := newSet[c.url]; !isNew {
			continue
		}
		if _, alreadySeen := p.seen[c.url]; alreadySeen {
			continue
		}
		p.seen[c.url] = struct{}{}
		freshCount++

		if !keepJob(c.job, p.gates) {
			gateFiltered++
			continue
		}
		queued = append(queued, dto.QueuedJob{URL: c.url, Card: c.job})
	}

	if gateFiltered > 0 {
		slog.Info("reject filter dropped jobs on page",
			slog.Int("filtered", gateFiltered),
			slog.Int("gates", len(p.gates)),
		)
	}

	if len(queued) == 0 {
		// Only stop if we've hit the known-URL frontier (all already in DB).
		// If URLs are fresh but gate-filtered, keep paginating.
		return freshCount == 0, nil
	}

	if err := p.q.EnqueueJobs(ctx, queued); err != nil {
		return false, err
	}

	slog.Info("enqueued page", slog.Int("count", len(queued)))
	return false, nil
}

func (o *Orchestrator) run(ctx context.Context, src sources.Source) error {
	cfg := src.Cfg()
	gates := o.loadGates(ctx, cfg.Name)
	if _, ok := src.(sources.DetailFetcher); ok {
		p := &htmlPath{db: o.db, q: o.q, gates: gates, seen: map[string]struct{}{}}
		return src.Iterate(ctx, p.onPage)
	}
	p := &atsPath{exporter: o.exporter, name: cfg.Name, gates: gates}
	return src.Iterate(ctx, p.onPage)
}

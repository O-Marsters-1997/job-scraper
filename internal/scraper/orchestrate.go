package scraper

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

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
	scorer       score.RelevanceScorer          // nil means no gate
	cfgDB        providers.SearchConfigProvider // nil means no gate
	scoreDB      providers.JobScoreProvider     // nil means no score writes
	exporter     JobExporter                    // nil means no ATS egress
	userID       string
	cr           *cron.Cron
	wg           sync.WaitGroup
	buildSources func(ctx context.Context) ([]sources.Source, error) // nil → use static srcs
	buildTarget  func(target dto.SourceTarget) []sources.Source      // nil → no on-demand scrape
}

func New(srcs []sources.Source, db providers.JobProvider, q queue.JobQueue) *Orchestrator {
	return &Orchestrator{srcs: srcs, db: db, q: q}
}

<<<<<<< HEAD
// RelevanceStore combines SearchConfigProvider and JobScoreProvider so callers pass db once.
type RelevanceStore interface {
	providers.SearchConfigProvider
	providers.JobScoreProvider
}

// WithRelevanceGate wires in a scorer and store so both ingestion paths filter
// jobs below cfg.RelevanceCutoff. ATS job scores are persisted via store.
func (o *Orchestrator) WithRelevanceGate(scorer score.RelevanceScorer, store RelevanceStore, userID string) {
=======
// WithSourceReloader wires in functions to reload sources from DB at each tick
// (buildAll) and to build a one-off source for on-demand scraping (buildOne).
// Calling this is optional; without it the static srcs slice is used.
func (o *Orchestrator) WithSourceReloader(
	buildAll func(ctx context.Context) ([]sources.Source, error),
	buildOne func(target dto.SourceTarget) []sources.Source,
) *Orchestrator {
	o.buildSources = buildAll
	o.buildTarget = buildOne
	return o
}

// WithRelevanceGate wires in a scorer, search config provider, and job score
// provider so that both ingestion paths filter jobs below cfg.RelevanceCutoff.
// Scores for ATS jobs are persisted via scoreDB.
func (o *Orchestrator) WithRelevanceGate(scorer score.RelevanceScorer, cfgDB providers.SearchConfigProvider, scoreDB providers.JobScoreProvider, userID string) *Orchestrator {
>>>>>>> 896a898 (support being able to add specific urls for wis)
	o.scorer = scorer
	o.cfgDB = store
	o.scoreDB = store
	o.userID = userID
}

// WithExporter wires in the egress port used by the ATS path.
func (o *Orchestrator) WithExporter(exp JobExporter) {
	o.exporter = exp
}

func (o *Orchestrator) Start(ctx context.Context) error {
<<<<<<< HEAD
	for _, src := range o.srcs {
		go o.runIfReady(ctx, src)
	}
=======
	// Run an initial scrape immediately in the background.
	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		o.tick(ctx)
	}()
>>>>>>> 896a898 (support being able to add specific urls for wis)

	o.cr = cron.New()

	// Schedule one reload-driven tick using the default scrape schedule.
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
		if err := o.run(ctx, src); err != nil {
			return err
		}
	}
	return nil
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

	last, ok, err := o.q.GetLastScraped(ctx, cfg.Name)
	if err != nil {
		log.Warn("could not read last scraped, proceeding",
			slog.Any("err", err),
		)
	}
	if ok && time.Since(last) < cfg.MinScrapeInterval {
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

type relevanceGate struct {
	enabled   bool
	scorer    score.RelevanceScorer
	searchCfg dto.SearchConfig
}

func (o *Orchestrator) loadGate(ctx context.Context, name string) relevanceGate {
	if o.scorer == nil || o.cfgDB == nil {
		return relevanceGate{}
	}
	cfg, err := o.cfgDB.GetSearchConfig(ctx, o.userID)
	if err != nil {
		slog.Warn("could not load search config, skipping relevance gate",
			slog.String("source", name),
			slog.Any("err", err),
		)
		return relevanceGate{}
	}
	return relevanceGate{enabled: true, scorer: o.scorer, searchCfg: cfg}
}

func (g relevanceGate) passes(job dto.Job) (int, bool) {
	if !g.enabled {
		return 0, true
	}
	s := g.scorer.Score(job, g.searchCfg)
	return s, s >= g.searchCfg.RelevanceCutoff
}

type atsPath struct {
	exporter JobExporter
	scoreDB  providers.JobScoreProvider
	name     string
	gate     relevanceGate
	userID   string
}

func (p *atsPath) onPage(ctx context.Context, jobs []dto.Job) (bool, error) {
	log := slog.With(slog.String("source", p.name))

	var valid []dto.Job
	var scores []int
	for _, j := range jobs {
		if j.URL == "" {
			continue
		}
		s, ok := p.gate.passes(j)
		if !ok {
			continue
		}
		valid = append(valid, j)
		scores = append(scores, s)
	}
	if len(valid) == 0 {
		return false, nil
	}

	if err := p.exporter.BulkExport(ctx, valid); err != nil {
		return false, err
	}

	if p.gate.enabled && p.scoreDB != nil {
		for i, j := range valid {
			if err := p.scoreDB.UpsertJobScoreRelevance(ctx, j.ID, p.userID, scores[i]); err != nil {
				log.Warn("could not write relevance score",
					slog.String("job_id", j.ID),
					slog.Any("err", err),
				)
			}
		}
	}

	log.Info("ats jobs published", slog.Int("count", len(valid)))
	return false, nil
}

type htmlPath struct {
	db   providers.JobProvider
	q    queue.JobQueue
	gate relevanceGate
	seen map[string]struct{}
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

	queued := make([]dto.QueuedJob, 0, len(candidates))
	for _, c := range candidates {
		if _, isNew := newSet[c.url]; !isNew {
			continue
		}
		if _, alreadySeen := p.seen[c.url]; alreadySeen {
			continue
		}
		p.seen[c.url] = struct{}{}

		qj := dto.QueuedJob{URL: c.url, Card: c.job}
		if p.gate.enabled {
			s, ok := p.gate.passes(c.job)
			if !ok {
				continue
			}
			qj.Relevance = s
		}
		queued = append(queued, qj)
	}

	if len(queued) == 0 {
		return true, nil
	}

	if err := p.q.EnqueueJobs(ctx, queued); err != nil {
		return false, err
	}

	slog.Info("enqueued page", slog.Int("count", len(queued)))
	return false, nil
}

func (o *Orchestrator) run(ctx context.Context, src sources.Source) error {
	cfg := src.Cfg()
	gate := o.loadGate(ctx, cfg.Name)
	if _, ok := src.(sources.DetailFetcher); ok {
		p := &htmlPath{db: o.db, q: o.q, gate: gate, seen: map[string]struct{}{}}
		return src.Iterate(ctx, p.onPage)
	}
	p := &atsPath{exporter: o.exporter, scoreDB: o.scoreDB, name: cfg.Name, gate: gate, userID: o.userID}
	return src.Iterate(ctx, p.onPage)
}

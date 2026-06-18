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
	"github.com/ollymarsters/job-scraper/internal/ingest"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/score"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

type Orchestrator struct {
	srcs     []sources.Source
	db       providers.JobProvider
	q        queue.JobQueue
	scorer   score.RelevanceScorer          // nil means no gate
	cfgDB    providers.SearchConfigProvider // nil means no gate
	scoreDB  providers.JobScoreProvider     // nil means no score writes
	ingester *ingest.Ingester               // nil means save-only (no scoring, no notify)
	userID   string
	cr       *cron.Cron
	wg       sync.WaitGroup
}

func New(srcs []sources.Source, db providers.JobProvider, q queue.JobQueue) *Orchestrator {
	return &Orchestrator{srcs: srcs, db: db, q: q}
}

// WithRelevanceGate wires in a scorer, search config provider, and job score
// provider so that both ingestion paths filter jobs below cfg.RelevanceCutoff.
// Scores for ATS jobs are persisted via scoreDB.
func (o *Orchestrator) WithRelevanceGate(scorer score.RelevanceScorer, cfgDB providers.SearchConfigProvider, scoreDB providers.JobScoreProvider, userID string) *Orchestrator {
	o.scorer = scorer
	o.cfgDB = cfgDB
	o.scoreDB = scoreDB
	o.userID = userID
	return o
}

// WithIngester wires in the ingest seam used by the ATS path. When set, Ingest
// replaces the direct db.Save call and also runs scoring and notifications.
func (o *Orchestrator) WithIngester(ing *ingest.Ingester) *Orchestrator {
	o.ingester = ing
	return o
}

func (o *Orchestrator) Start(ctx context.Context) error {
	go func() {
		var wg sync.WaitGroup
		for _, src := range o.srcs {
			wg.Add(1)
			go func(src sources.Source) {
				defer wg.Done()
				o.runIfReady(ctx, src)
			}(src)
		}
		wg.Wait()
	}()

	o.cr = cron.New()

	for _, src := range o.srcs {
		cfg := src.Cfg()
		if _, err := o.cr.AddFunc(cfg.Schedule, func() {
			slog.Info("cron: starting scrape",
				slog.String("source", cfg.Name),
			)
			o.wg.Add(1)
			defer o.wg.Done()
			o.runIfReady(ctx, src)
		}); err != nil {
			o.cr.Stop()
			return fmt.Errorf("schedule %s (%s): %w", cfg.Name, cfg.Schedule, err)
		}
		slog.Info("cron: scheduled",
			slog.String("source", cfg.Name),
			slog.String("schedule", cfg.Schedule),
		)
	}

	o.cr.Start()
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
	scorer  score.RelevanceScorer
	cfg     dto.SearchConfig
	scoreDB providers.JobScoreProvider
	userID  string
	enabled bool
}

// keep returns (score, true) if the job passes the gate, or (0, false) to drop it.
// When gate is disabled, always returns (0, true).
func (g *relevanceGate) keep(job dto.Job) (int, bool) {
	if !g.enabled {
		return 0, true
	}
	s := g.scorer.Score(job, g.cfg)
	if s < g.cfg.RelevanceCutoff {
		return 0, false
	}
	return s, true
}

func (o *Orchestrator) loadGate(ctx context.Context) relevanceGate {
	if o.scorer == nil || o.cfgDB == nil {
		return relevanceGate{}
	}
	cfg, err := o.cfgDB.GetSearchConfig(ctx, o.userID)
	if err != nil {
		slog.Warn("could not load search config, skipping relevance gate", slog.Any("err", err))
		return relevanceGate{}
	}
	return relevanceGate{
		scorer:  o.scorer,
		cfg:     cfg,
		scoreDB: o.scoreDB,
		userID:  o.userID,
		enabled: true,
	}
}

type atsPath struct {
	db      providers.JobProvider
	ing     *ingest.Ingester
	scoreDB providers.JobScoreProvider
	name    string
	gate    relevanceGate
}

func (a *atsPath) onPage(ctx context.Context, jobs []dto.Job) (bool, error) {
	log := slog.With(slog.String("source", a.name))

	type scored struct {
		job   dto.Job
		score int
	}
	survivors := make([]scored, 0, len(jobs))
	for _, j := range jobs {
		if j.URL == "" {
			continue
		}
		s, ok := a.gate.keep(j)
		if !ok {
			continue
		}
		survivors = append(survivors, scored{job: j, score: s})
	}
	if len(survivors) == 0 {
		return false, nil
	}

	validJobs := make([]dto.Job, len(survivors))
	for i, s := range survivors {
		validJobs[i] = s.job
	}
	if a.ing != nil {
		if err := a.ing.Ingest(ctx, validJobs); err != nil {
			return false, err
		}
	} else if err := a.db.Save(ctx, validJobs); err != nil {
		return false, err
	}

	if a.gate.enabled && a.scoreDB != nil {
		for _, s := range survivors {
			if err := a.scoreDB.UpsertJobScoreRelevance(ctx, s.job.ID, a.gate.userID, s.score); err != nil {
				log.Warn("could not write relevance score",
					slog.String("job_id", s.job.ID),
					slog.Any("err", err),
				)
			}
		}
	}

	log.Info("ats jobs saved", slog.Int("count", len(validJobs)))
	return false, nil
}

type htmlPath struct {
	db   providers.JobProvider
	q    queue.JobQueue
	df   sources.DetailFetcher
	gate relevanceGate
	seen map[string]struct{}
	name string
}

func (h *htmlPath) onPage(ctx context.Context, jobs []dto.Job) (bool, error) {
	log := slog.With(slog.String("source", h.name))

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

	newURLs, err := h.db.NewURLs(ctx, urls)
	if err != nil {
		log.Warn("filter failed, using all URLs", slog.Any("err", err))
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
		if _, alreadySeen := h.seen[c.url]; alreadySeen {
			continue
		}
		h.seen[c.url] = struct{}{}

		qj := dto.QueuedJob{URL: c.url, Card: c.job}
		s, ok := h.gate.keep(c.job)
		if !ok {
			continue
		}
		qj.Relevance = s
		queued = append(queued, qj)
	}

	if len(queued) == 0 {
		return true, nil
	}

	if err := h.q.EnqueueJobs(ctx, queued); err != nil {
		return false, err
	}

	log.Info("enqueued page", slog.Int("count", len(queued)))
	return false, nil
}

func (o *Orchestrator) run(ctx context.Context, src sources.Source) error {
	gate := o.loadGate(ctx)
	if df, ok := src.(sources.DetailFetcher); ok {
		return src.Iterate(ctx, (&htmlPath{db: o.db, q: o.q, df: df, gate: gate, seen: map[string]struct{}{}, name: src.Cfg().Name}).onPage)
	}
	return src.Iterate(ctx, (&atsPath{db: o.db, ing: o.ingester, scoreDB: o.scoreDB, name: src.Cfg().Name, gate: gate}).onPage)
}

package scraper

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
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
	cfgDB        providers.SearchConfigProvider  // nil means no gate
	scoreDB      providers.JobScoreProvider      // nil means no score writes
	ingestScorer *score.IngestScorer             // nil means no suitability scoring
	userID       string
	cr           *cron.Cron
	wg           sync.WaitGroup
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

// WithSuitabilityScorer wires in a score-on-ingest helper that writes a
// suitability score after every ATS job save.
func (o *Orchestrator) WithSuitabilityScorer(is *score.IngestScorer) *Orchestrator {
	o.ingestScorer = is
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

func (o *Orchestrator) run(ctx context.Context, src sources.Source) error {
	name := src.Cfg().Name
	log := slog.With(slog.String("source", name))
	seen := make(map[string]struct{})

	// Load search config once per run (only when gating is enabled).
	var searchCfg dto.SearchConfig
	gating := o.scorer != nil && o.cfgDB != nil
	if gating {
		cfg, err := o.cfgDB.GetSearchConfig(ctx, o.userID)
		if err != nil {
			log.Warn("could not load search config, skipping relevance gate", slog.Any("err", err))
			gating = false
		} else {
			searchCfg = cfg
		}
	}

	return src.Iterate(ctx, func(ctx context.Context, jobs []dto.Job) (bool, error) {
		if !src.NeedsDetail() {
			// ATS path: jobs arrive fully-populated — apply relevance gate then save.
			type scored struct {
				job   dto.Job
				score int
			}
			survivors := make([]scored, 0, len(jobs))
			for _, j := range jobs {
				if j.URL == "" {
					continue
				}
				if gating {
					s := o.scorer.Score(j, searchCfg)
					if s < searchCfg.RelevanceCutoff {
						continue
					}
					survivors = append(survivors, scored{job: j, score: s})
				} else {
					survivors = append(survivors, scored{job: j})
				}
			}
			if len(survivors) == 0 {
				return false, nil
			}

			validJobs := make([]dto.Job, len(survivors))
			for i, s := range survivors {
				validJobs[i] = s.job
			}
			if err := o.db.Save(ctx, validJobs); err != nil {
				return false, err
			}

			if o.ingestScorer != nil {
				for _, s := range survivors {
					o.ingestScorer.ScoreAndSave(ctx, s.job)
				}
			}

			if gating && o.scoreDB != nil {
				for _, s := range survivors {
					if err := o.scoreDB.UpsertJobScoreRelevance(ctx, s.job.ID, o.userID, s.score); err != nil {
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

		// HTML path (NeedsDetail=true): extract URLs, filter new, enqueue with payload.
		type partial struct {
			url string
			job dto.Job
		}
		candidates := make([]partial, 0, len(jobs))
		for _, j := range jobs {
			if j.URL != "" {
				candidates = append(candidates, partial{url: j.URL, job: j})
			}
		}

		urls := make([]string, len(candidates))
		for i, c := range candidates {
			urls[i] = c.url
		}

		newURLs, err := o.db.NewURLs(ctx, urls)
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
			if _, alreadySeen := seen[c.url]; alreadySeen {
				continue
			}
			seen[c.url] = struct{}{}

			qj := dto.QueuedJob{URL: c.url, Card: c.job}
			if gating {
				s := o.scorer.Score(c.job, searchCfg)
				if s < searchCfg.RelevanceCutoff {
					continue
				}
				qj.Relevance = s
			}
			queued = append(queued, qj)
		}

		if len(queued) == 0 {
			return true, nil
		}

		if err := o.q.EnqueueJobs(ctx, queued); err != nil {
			return false, err
		}

		log.Info("enqueued page", slog.Int("count", len(queued)))
		return false, nil
	})
}

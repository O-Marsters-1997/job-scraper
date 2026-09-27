package scraper

import (
	"context"
	"fmt"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/candidates"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

type Orchestrator struct {
	db providers.JobProvider
	q  interface {
		EnqueueJobs(context.Context, []dto.QueuedJob) error
	}
	cfgDB       providers.SearchConfigProvider
	buildTarget func(dto.SourceTarget) []sources.Source
	candidates  *candidates.Service
}

func New(db providers.JobProvider, q interface {
	EnqueueJobs(context.Context, []dto.QueuedJob) error
}) *Orchestrator {
	return &Orchestrator{db: db, q: q}
}

func (o *Orchestrator) WithSourceBuilder(build func(dto.SourceTarget) []sources.Source) *Orchestrator {
	o.buildTarget = build
	return o
}

func (o *Orchestrator) WithRejectFilter(cfgDB providers.SearchConfigProvider) {
	o.cfgDB = cfgDB
}

func (o *Orchestrator) WithCandidates(store candidates.Store) *Orchestrator {
	o.candidates = candidates.New(store, o.q)
	return o
}

func (o *Orchestrator) searchConfig(ctx context.Context, target dto.SourceTarget) (dto.SearchConfig, error) {
	config := dto.SearchConfig{UserID: target.UserID}
	if o.cfgDB != nil {
		stored, err := o.cfgDB.GetSearchConfig(ctx, target.UserID)
		if err != nil && err != data.ErrNotFound {
			return config, err
		}
		if err == nil {
			config = stored
		}
	}
	return config, nil
}

func rewriteCards(cards []dto.Job) {
	for i := range cards {
		if detect.Detect(cards[i].URL) == detect.Aggregator {
			if rewritten, _, ok := detect.RewriteToATS(cards[i].URL); ok {
				cards[i].URL = rewritten
			}
		}
	}
}

func (o *Orchestrator) ScrapePage(ctx context.Context, target dto.SourceTarget, cursor string) (string, error) {
	if o.buildTarget == nil || o.candidates == nil {
		return "", fmt.Errorf("discovery page processor unavailable")
	}
	srcs := o.buildTarget(target)
	if len(srcs) != 1 {
		return "", fmt.Errorf("expected one source for target %s", target.ID)
	}
	config, err := o.searchConfig(ctx, target)
	if err != nil {
		return "", err
	}
	var cards []dto.Job
	var next string
	if fetcher, ok := srcs[0].(sources.PageFetcher); ok {
		cards, next, err = fetcher.FetchPage(ctx, cursor)
	} else if cursor == "" {
		err = srcs[0].Iterate(ctx, func(_ context.Context, page []dto.Job) (bool, error) {
			cards = append(cards, page...)
			return false, nil
		})
	} else {
		return "", fmt.Errorf("source %s has no page cursor", target.Source)
	}
	if err != nil {
		return "", err
	}
	rewriteCards(cards)
	if len(cards) > 0 {
		urls := make([]string, 0, len(cards))
		for _, card := range cards {
			if card.URL != "" {
				urls = append(urls, card.URL)
			}
		}
		newURLs, err := o.db.NewURLs(ctx, urls)
		if err != nil {
			return "", err
		}
		if len(newURLs) == 0 {
			next = ""
		}
	}
	return next, o.candidates.CapturePage(ctx, target, cards, config)
}

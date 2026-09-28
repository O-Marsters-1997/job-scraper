package scraper

import (
	"context"
	"errors"
	"fmt"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

// SearchConfigReader reads a user's Search Config from the scoring context.
type SearchConfigReader interface {
	SearchConfig(ctx context.Context, userID string) (dto.SearchConfig, error)
}

// URLChecker filters URLs already known to the jobsearch catalog.
type URLChecker interface {
	NewURLs(ctx context.Context, urls []string) ([]string, error)
}

// CandidateCapturer saves a scraped page's cards as candidates and assesses
// them against a Search Config; jobsearch's candidates facade satisfies it.
type CandidateCapturer interface {
	CapturePage(ctx context.Context, target dto.SourceTarget, cards []dto.Job, config dto.SearchConfig) error
}

type Orchestrator struct {
	db          URLChecker
	cfgDB       SearchConfigReader
	buildTarget func(dto.SourceTarget) (sources.Source, bool)
	candidates  CandidateCapturer
}

func New(db URLChecker) *Orchestrator {
	return &Orchestrator{db: db}
}

func (o *Orchestrator) WithSourceBuilder(build func(dto.SourceTarget) (sources.Source, bool)) *Orchestrator {
	o.buildTarget = build
	return o
}

func (o *Orchestrator) WithRejectFilter(cfgDB SearchConfigReader) {
	o.cfgDB = cfgDB
}

func (o *Orchestrator) WithCandidates(capturer CandidateCapturer) *Orchestrator {
	o.candidates = capturer
	return o
}

func (o *Orchestrator) searchConfig(ctx context.Context, target dto.SourceTarget) (dto.SearchConfig, error) {
	config := dto.SearchConfig{UserID: target.UserID}
	if o.cfgDB != nil {
		stored, err := o.cfgDB.SearchConfig(ctx, target.UserID)
		if err != nil && !errors.Is(err, data.ErrNotFound) {
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
	src, ok := o.buildTarget(target)
	if !ok {
		return "", fmt.Errorf("unsupported source for target %s", target.ID)
	}
	config, err := o.searchConfig(ctx, target)
	if err != nil {
		return "", err
	}
	cards, next, err := src.FetchPage(ctx, cursor)
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

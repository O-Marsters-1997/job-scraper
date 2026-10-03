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

func New(db URLChecker, cfgDB SearchConfigReader, buildTarget func(dto.SourceTarget) (sources.Source, bool), candidates CandidateCapturer) *Orchestrator {
	return &Orchestrator{db: db, cfgDB: cfgDB, buildTarget: buildTarget, candidates: candidates}
}

func (o *Orchestrator) searchConfig(ctx context.Context, target dto.SourceTarget) (dto.SearchConfig, error) {
	stored, err := o.cfgDB.SearchConfig(ctx, target.UserID)
	if errors.Is(err, data.ErrNotFound) {
		return dto.SearchConfig{UserID: target.UserID}, nil
	}
	return stored, err
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
		if len(newURLs) == 0 && src.Cfg().NewestFirst {
			next = ""
		}
	}
	return next, o.candidates.CapturePage(ctx, target, cards, config)
}

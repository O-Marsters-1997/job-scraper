// Package builder is separate from sources because sources sub-packages
// (e.g. sources/greenhouse) import sources, which would create a cycle if
// BuildSources were defined there.
package builder

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
	"github.com/ollymarsters/job-scraper/internal/sources/ashby"
	"github.com/ollymarsters/job-scraper/internal/sources/greenhouse"
	"github.com/ollymarsters/job-scraper/internal/sources/indeed"
	"github.com/ollymarsters/job-scraper/internal/sources/lever"
	"github.com/ollymarsters/job-scraper/internal/sources/linkedin"
	"github.com/ollymarsters/job-scraper/internal/sources/personio"
	"github.com/ollymarsters/job-scraper/internal/sources/recruitee"
	"github.com/ollymarsters/job-scraper/internal/sources/wis"
	"github.com/ollymarsters/job-scraper/internal/sources/workable"
)

// BuildSources constructs the live Source set from enabled targets. boardDone,
// if non-nil, is called with (source name, board token) whenever a board is
// successfully scraped, so the caller can record per-target freshness. Pass
// nil for one-off scrapes (e.g. scrape-now) that must not affect scheduling.
func BuildSources(targets []dto.SourceTarget, boardDone func(ctx context.Context, source, value string)) []sources.Source {
	boards := make(map[string][]string)     // source name → deduped board tokens
	seen := make(map[string]bool)           // "source\x00token" → already added
	urlSources := make(map[string][]string) // source name → configured search URLs
	var wisSearches []wis.Search
	var linkedinSearches []linkedin.Search

	for _, t := range targets {
		if !t.Enabled {
			continue
		}
		if sources.IsFilterSource(t.Source) {
			switch t.Source {
			case "wis":
				wisSearches = append(wisSearches, wis.Search{
					Keywords: t.Value,
					Region:   t.Filters["region"],
				})
			case "linkedin":
				linkedinSearches = append(linkedinSearches, linkedin.Search{
					Keywords: t.Value,
					Location: t.Filters["location"],
				})
			}
			continue
		}
		_, isURL, ok := sources.LookupSource(t.Source)
		if !ok {
			continue
		}
		if isURL {
			urlSources[t.Source] = append(urlSources[t.Source], t.Value)
			continue
		}
		key := t.Source + "\x00" + t.Value
		if seen[key] {
			continue
		}
		seen[key] = true
		boards[t.Source] = append(boards[t.Source], t.Value)
	}

	var srcs []sources.Source

	if len(wisSearches) > 0 {
		srcs = append(srcs, wis.New(wis.Config{Searches: wisSearches}))
	}
	if len(linkedinSearches) > 0 {
		srcs = append(srcs, linkedin.New(linkedin.Config{Searches: linkedinSearches}))
	}

	withDone := func(name string, src *sources.BoardSource) *sources.BoardSource {
		if boardDone == nil {
			return src
		}
		return src.WithDone(func(ctx context.Context, token string) {
			boardDone(ctx, name, token)
		})
	}

	if tokens := boards["greenhouse"]; len(tokens) > 0 {
		srcs = append(srcs, withDone("greenhouse", greenhouse.New(greenhouse.Config{Boards: tokens})))
	}
	if tokens := boards["lever"]; len(tokens) > 0 {
		srcs = append(srcs, withDone("lever", lever.New(lever.Config{Boards: tokens})))
	}
	if tokens := boards["ashby"]; len(tokens) > 0 {
		srcs = append(srcs, withDone("ashby", ashby.New(ashby.Config{Boards: tokens})))
	}
	if tokens := boards["workable"]; len(tokens) > 0 {
		srcs = append(srcs, withDone("workable", workable.New(workable.Config{Boards: tokens})))
	}
	if tokens := boards["recruitee"]; len(tokens) > 0 {
		srcs = append(srcs, withDone("recruitee", recruitee.New(recruitee.Config{Boards: tokens})))
	}
	if tokens := boards["personio"]; len(tokens) > 0 {
		srcs = append(srcs, withDone("personio", personio.New(personio.Config{Boards: tokens})))
	}
	if urls := urlSources["indeed"]; len(urls) > 0 {
		srcs = append(srcs, indeed.New(indeed.Config{URLs: urls}))
	}

	return srcs
}

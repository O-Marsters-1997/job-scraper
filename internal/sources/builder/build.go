// Package builder is separate from sources because sources sub-packages
// (e.g. sources/greenhouse) import sources, which would create a cycle if
// BuildSources were defined there.
package builder

import (
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

func BuildSources(targets []dto.SourceTarget) []sources.Source {
	boards := make(map[string][]string) // source name → board tokens
	urlSources := make(map[string]bool)
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
			urlSources[t.Source] = true
		} else {
			boards[t.Source] = append(boards[t.Source], t.Value)
		}
	}

	var srcs []sources.Source

	if len(wisSearches) > 0 {
		srcs = append(srcs, wis.New(wis.Config{Searches: wisSearches}))
	}
	if len(linkedinSearches) > 0 {
		srcs = append(srcs, linkedin.New(linkedin.Config{Searches: linkedinSearches}))
	}

	if tokens := boards["greenhouse"]; len(tokens) > 0 {
		srcs = append(srcs, greenhouse.New(greenhouse.Config{Boards: tokens}))
	}
	if tokens := boards["lever"]; len(tokens) > 0 {
		srcs = append(srcs, lever.New(lever.Config{Boards: tokens}))
	}
	if tokens := boards["ashby"]; len(tokens) > 0 {
		srcs = append(srcs, ashby.New(ashby.Config{Boards: tokens}))
	}
	if tokens := boards["workable"]; len(tokens) > 0 {
		srcs = append(srcs, workable.New(workable.Config{Boards: tokens}))
	}
	if tokens := boards["recruitee"]; len(tokens) > 0 {
		srcs = append(srcs, recruitee.New(recruitee.Config{Boards: tokens}))
	}
	if tokens := boards["personio"]; len(tokens) > 0 {
		srcs = append(srcs, personio.New(personio.Config{Boards: tokens}))
	}
	if urlSources["indeed"] {
		srcs = append(srcs, indeed.New())
	}

	return srcs
}

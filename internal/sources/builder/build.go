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
	"github.com/ollymarsters/job-scraper/internal/sources/registry"
	"github.com/ollymarsters/job-scraper/internal/sources/remoteok"
	"github.com/ollymarsters/job-scraper/internal/sources/remotive"
	"github.com/ollymarsters/job-scraper/internal/sources/wis"
	"github.com/ollymarsters/job-scraper/internal/sources/workable"
)

func BuildSources(targets []dto.SourceTarget) []sources.Source {
	boards := make(map[string][]string)     // source name → deduped board tokens
	seen := make(map[string]bool)           // "source\x00token" → already added
	urlSources := make(map[string][]string) // source name → configured search URLs
	var wisSearches []wis.Search
	var linkedinSearches []linkedin.Search
	var remoteokKeywords []string
	var remotiveKeywords []string

	for _, t := range targets {
		if !t.Enabled {
			continue
		}
		if registry.IsFilterSource(t.Source) {
			switch t.Source {
			case "wis":
				wisSearches = append(wisSearches, wis.Search{
					Keywords: t.Value,
					Region:   t.Filters["region"],
				})
			case "linkedin":
				linkedinSearches = append(linkedinSearches, linkedin.Search{
					Keywords:    t.Value,
					Location:    t.Filters["location"],
					CompanyID:   t.Filters["company_id"],
					Recency:     t.Filters["recency"],
					Arrangement: t.Filters["arrangement"],
					Experience:  t.Filters["experience"],
					JobType:     t.Filters["job_type"],
					GeoID:       t.Filters["geo_id"],
					Distance:    t.Filters["distance"],
					SalaryBand:  t.Filters["salary_band"],
				})
			case "remoteok":
				remoteokKeywords = append(remoteokKeywords, t.Value)
			case "remotive":
				remotiveKeywords = append(remotiveKeywords, t.Value)
			}
			continue
		}
		_, isURL, ok := registry.LookupSource(t.Source)
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
	if len(remoteokKeywords) > 0 {
		srcs = append(srcs, remoteok.New(remoteok.Config{Keywords: remoteokKeywords}))
	}
	if len(remotiveKeywords) > 0 {
		srcs = append(srcs, remotive.New(remotive.Config{Keywords: remotiveKeywords}))
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
	if urls := urlSources["indeed"]; len(urls) > 0 {
		srcs = append(srcs, indeed.New(indeed.Config{URLs: urls}))
	}

	return srcs
}

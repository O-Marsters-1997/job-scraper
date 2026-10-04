// Package builder is separate from sources because sources sub-packages
// (e.g. sources/greenhouse) import sources, which would create a cycle if
// BuildSource were defined there.
package builder

import (
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/ashby"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/greenhouse"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/hibob"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/indeed"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/lever"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/linkedin"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/personio"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/pinpoint"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/recruitee"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/remoteok"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/remotive"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/teamtailor"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/wis"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/workable"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/wttj"
)

type entry struct {
	build  func(dto.SourceTarget) sources.Source
	detail sources.DetailFetcher
}

func boardEntry(newSource func(token string) sources.Source) entry {
	return entry{build: func(t dto.SourceTarget) sources.Source { return newSource(t.Value) }}
}

var registry = map[string]entry{
	"greenhouse": boardEntry(func(v string) sources.Source { return greenhouse.New(v) }),
	"lever":      boardEntry(func(v string) sources.Source { return lever.New(v) }),
	"ashby":      boardEntry(func(v string) sources.Source { return ashby.New(v) }),
	"workable":   boardEntry(func(v string) sources.Source { return workable.New(v) }),
	"recruitee":  boardEntry(func(v string) sources.Source { return recruitee.New(v) }),
	"personio":   boardEntry(func(v string) sources.Source { return personio.New(v) }),
	"pinpoint":   boardEntry(func(v string) sources.Source { return pinpoint.New(v) }),
	"teamtailor": boardEntry(func(v string) sources.Source { return teamtailor.New(v) }),
	"hibob":      boardEntry(func(v string) sources.Source { return hibob.New(v) }),
	"wttj":       {build: func(t dto.SourceTarget) sources.Source { return wttj.New(t.Value, time.Now) }},
	"indeed":     {build: func(t dto.SourceTarget) sources.Source { return indeed.New(t.Value, t.Filters) }},
	"remoteok":   {build: func(t dto.SourceTarget) sources.Source { return remoteok.New(t.Value) }},
	"remotive":   {build: func(t dto.SourceTarget) sources.Source { return remotive.New(t.Value) }},
	"wis": {
		build: func(t dto.SourceTarget) sources.Source {
			return wis.New(wis.Search{
				Keywords: t.Value,
				Filters:  t.Filters,
				Recency:  wis.Recency(t.LastSucceededAt, time.Now()),
			})
		},
		detail: wis.New(wis.Search{}),
	},
	"linkedin": {
		build: func(t dto.SourceTarget) sources.Source {
			return linkedin.New(t.Value, t.Filters, linkedin.Recency(t.Filters["recency"], t.LastSucceededAt, time.Now()))
		},
		detail: linkedin.New("", nil, ""),
	},
}

// BuildSource builds the Source for one enabled target. It returns false when
// the target is disabled or its source is unknown.
func BuildSource(t dto.SourceTarget) (sources.Source, bool) {
	e, ok := registry[t.Source]
	if !ok || !t.Enabled {
		return nil, false
	}
	return e.build(t), true
}

// Detailers returns the detail fetcher of every source that has one.
func Detailers() map[string]sources.DetailFetcher {
	out := map[string]sources.DetailFetcher{}
	for name, e := range registry {
		if e.detail != nil {
			out[name] = e.detail
		}
	}
	return out
}

// CardComplete returns the sources whose listing card is already the full job.
func CardComplete() map[string]bool {
	out := map[string]bool{}
	for name := range registry {
		if sourcespec.CardComplete(name) {
			out[name] = true
		}
	}
	return out
}

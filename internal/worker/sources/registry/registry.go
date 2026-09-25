package registry

import (
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/greenhouse"
)

type pageSourceEntry struct {
	requestGap    time.Duration
	newPageSource func(string) sources.PageSource
}

var pageSources = map[string]pageSourceEntry{
	"greenhouse": {requestGap: 2 * time.Second, newPageSource: func(token string) sources.PageSource {
		return greenhouse.New(greenhouse.Config{Boards: []string{token}})
	}},
}

// Entry contains a constructed page source and its execution policy.
type Entry struct {
	Name       string
	Role       string
	RequestGap time.Duration
	Source     sources.PageSource
	Details    sources.PageDetailFetcher
}

func Open(target dto.SourceTarget) (Entry, error) {
	registration, ok := pageSources[target.Source]
	role, known := sourcespec.SourceRole(target.Source)
	if !ok || !known {
		return Entry{}, fmt.Errorf("unsupported page source %q", target.Source)
	}
	if !target.Enabled || len(target.Filters) != 0 || !sourcespec.ValidBoardToken(target.Value) {
		return Entry{}, fmt.Errorf("invalid %s board configuration", target.Source)
	}
	src := registration.newPageSource(target.Value)
	entry := Entry{Name: target.Source, Role: role, RequestGap: registration.requestGap, Source: src}
	entry.Details, _ = src.(sources.PageDetailFetcher)
	return entry, nil
}

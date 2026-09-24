package catalog

import (
	"fmt"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
	"github.com/ollymarsters/job-scraper/internal/sources/greenhouse"
)

type registration struct {
	role       string
	requestGap time.Duration
	newSource  func(string) sources.PageSource
}

// Entry contains a constructed page source and its execution policy.
type Entry struct {
	Name       string
	Role       string
	RequestGap time.Duration
	Source     sources.PageSource
	Details    sources.PageDetailFetcher
}

var registrations = map[string]registration{
	"greenhouse": {
		role:       sources.RoleATS,
		requestGap: 2 * time.Second,
		newSource: func(token string) sources.PageSource {
			return greenhouse.New(greenhouse.Config{Boards: []string{token}})
		},
	},
}

// Open constructs a page source for one saved target.
func Open(target dto.SourceTarget) (Entry, error) {
	registration, ok := registrations[target.Source]
	if !ok {
		return Entry{}, fmt.Errorf("unsupported page source %q", target.Source)
	}
	if !target.Enabled || len(target.Filters) != 0 || !ValidBoardToken(target.Value) {
		return Entry{}, fmt.Errorf("invalid %s board configuration", target.Source)
	}
	src := registration.newSource(target.Value)
	entry := Entry{Name: target.Source, Role: registration.role, RequestGap: registration.requestGap, Source: src}
	entry.Details, _ = src.(sources.PageDetailFetcher)
	return entry, nil
}

// ValidBoardToken reports whether a board token can be used as one URL path segment.
func ValidBoardToken(token string) bool {
	if token == "" || token == "." || token == ".." {
		return false
	}
	for _, r := range token {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._-", r) {
			return false
		}
	}
	return true
}

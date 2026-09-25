package registry

import (
	"fmt"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
	"github.com/ollymarsters/job-scraper/internal/sources/greenhouse"
)

type sourceKind int

const (
	kindBoard sourceKind = iota
	kindURL
	kindFilter
)

func (k sourceKind) string() string {
	switch k {
	case kindBoard:
		return "board"
	case kindURL:
		return "url"
	case kindFilter:
		return "filter"
	}
	return ""
}

// Source roles classify a source by purpose, orthogonal to sourceKind (which
// describes value shape). RoleATS sources are known-company boards re-checked
// periodically; RoleDiscovery sources are search surfaces (aggregators, HTML boards).
const (
	RoleATS       = "ats"
	RoleDiscovery = "discovery"
)

// FilterField describes a structured filter parameter accepted by a kindFilter source.
type FilterField struct {
	Name     string `json:"name"`
	Label    string `json:"label"`
	Required bool   `json:"required"`
}

// SourceInfo is the serialisable view of a registry entry, suitable for API responses.
type SourceInfo struct {
	Name      string        `json:"name"`
	Label     string        `json:"label"`
	Kind      string        `json:"kind"` // "board" | "url" | "filter"
	Role      string        `json:"role"` // "ats" | "discovery"
	URLPrefix string        `json:"url_prefix"`
	Filters   []FilterField `json:"filters"` // non-empty only for kindFilter sources
}

type registryEntry struct {
	name          string
	label         string
	kind          sourceKind
	role          string
	urlPrefix     string
	filters       []FilterField
	requestGap    time.Duration
	newPageSource func(string) sources.PageSource
}

var entries = []registryEntry{
	{name: "greenhouse", label: "Greenhouse", kind: kindBoard, role: RoleATS, urlPrefix: "https://boards.greenhouse.io", requestGap: 2 * time.Second, newPageSource: func(token string) sources.PageSource {
		return greenhouse.New(greenhouse.Config{Boards: []string{token}})
	}},
	{name: "lever", label: "Lever", kind: kindBoard, role: RoleATS, urlPrefix: "https://jobs.lever.co"},
	{name: "ashby", label: "Ashby", kind: kindBoard, role: RoleATS, urlPrefix: "https://jobs.ashbyhq.com"},
	{name: "workable", label: "Workable", kind: kindBoard, role: RoleATS, urlPrefix: "https://apply.workable.com"},
	{name: "recruitee", label: "Recruitee", kind: kindBoard, role: RoleATS, urlPrefix: "https://recruitee.com"},
	{name: "personio", label: "Personio", kind: kindBoard, role: RoleATS, urlPrefix: "https://personio.de"},
	{name: "wis", label: "Work in Startups", kind: kindFilter, role: RoleDiscovery, urlPrefix: "https://workinstartups.com", filters: []FilterField{
		{Name: "region", Label: "Region", Required: false},
	}},
	{name: "linkedin", label: "LinkedIn", kind: kindFilter, role: RoleDiscovery, urlPrefix: "https://www.linkedin.com/jobs", filters: []FilterField{
		{Name: "location", Label: "Location", Required: false},
		{Name: "company_id", Label: "Company ID", Required: false},
		{Name: "recency", Label: "Recency", Required: false},
		{Name: "arrangement", Label: "Work Arrangement", Required: false},
		{Name: "experience", Label: "Experience Level", Required: false},
		{Name: "job_type", Label: "Job Type", Required: false},
		{Name: "geo_id", Label: "Geo ID", Required: false},
		{Name: "distance", Label: "Distance", Required: false},
		{Name: "salary_band", Label: "Salary Band", Required: false},
	}},
	{name: "indeed", label: "Indeed", kind: kindURL, role: RoleDiscovery, urlPrefix: "https://www.indeed.com"},
	{name: "remoteok", label: "RemoteOK", kind: kindFilter, role: RoleDiscovery, urlPrefix: "https://remoteok.com"},
	{name: "remotive", label: "Remotive", kind: kindFilter, role: RoleDiscovery, urlPrefix: "https://remotive.com"},
}

func Sources() []SourceInfo {
	infos := make([]SourceInfo, len(entries))
	for i, e := range entries {
		filters := e.filters
		if filters == nil {
			filters = []FilterField{}
		}
		infos[i] = SourceInfo{
			Name:      e.name,
			Label:     e.label,
			Kind:      e.kind.string(),
			Role:      e.role,
			URLPrefix: e.urlPrefix,
			Filters:   filters,
		}
	}
	return infos
}

func findEntry(name string) (registryEntry, bool) {
	for _, e := range entries {
		if e.name == name {
			return e, true
		}
	}
	return registryEntry{}, false
}

// LookupSource returns the URL prefix and whether the source is URL-based (kindURL), plus
// whether the source name is known at all. kindFilter sources return isURL=false.
func LookupSource(name string) (urlPrefix string, isURL bool, ok bool) {
	e, ok := findEntry(name)
	if !ok {
		return "", false, false
	}
	return e.urlPrefix, e.kind == kindURL, true
}

// LookupFilterFields returns the declared filter fields for a kindFilter source,
// plus whether the source is a filter source at all.
func LookupFilterFields(name string) ([]FilterField, bool) {
	e, ok := findEntry(name)
	if !ok || e.kind != kindFilter {
		return nil, false
	}
	return e.filters, true
}

// SourceRole returns the role (RoleATS | RoleDiscovery) for a source, plus whether
// the source name is known.
func SourceRole(name string) (string, bool) {
	e, ok := findEntry(name)
	if !ok {
		return "", false
	}
	return e.role, true
}

func IsFilterSource(name string) bool {
	e, ok := findEntry(name)
	return ok && e.kind == kindFilter
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
	registration, ok := findEntry(target.Source)
	if !ok || registration.newPageSource == nil {
		return Entry{}, fmt.Errorf("unsupported page source %q", target.Source)
	}
	if !target.Enabled || len(target.Filters) != 0 || !ValidBoardToken(target.Value) {
		return Entry{}, fmt.Errorf("invalid %s board configuration", target.Source)
	}
	src := registration.newPageSource(target.Value)
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

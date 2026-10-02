package sourcespec

import (
	"slices"
	"strings"
)

type sourceKind string

const (
	kindBoard  sourceKind = "board"
	kindURL    sourceKind = "url"
	kindFilter sourceKind = "filter"
)

// Source roles classify a source by purpose, orthogonal to sourceKind (which
// describes value shape). RoleATS sources are known-company boards re-checked
// periodically; RoleDiscovery sources are search surfaces (aggregators, HTML boards).
const (
	RoleATS       = "ats"
	RoleDiscovery = "discovery"
)

// FilterField describes a structured filter parameter accepted by a kindFilter source.
type FilterField struct {
	Name     string         `json:"name"`
	Param    string         `json:"-"`
	Numeric  bool           `json:"-"`
	Label    string         `json:"label"`
	Required bool           `json:"required"`
	Options  []FilterOption `json:"options,omitempty"` // non-empty for enumerated fields
}

// FilterOption is one permitted value of an enumerated FilterField.
type FilterOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
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
	name         string
	label        string
	kind         sourceKind
	role         string
	urlPrefix    string
	filters      []FilterField
	cardComplete bool
}

var entries = []registryEntry{
	{name: "greenhouse", label: "Greenhouse", kind: kindBoard, role: RoleATS, urlPrefix: "https://boards.greenhouse.io"},
	{name: "lever", label: "Lever", kind: kindBoard, role: RoleATS, urlPrefix: "https://jobs.lever.co"},
	{name: "ashby", label: "Ashby", kind: kindBoard, role: RoleATS, urlPrefix: "https://jobs.ashbyhq.com"},
	{name: "workable", label: "Workable", kind: kindBoard, role: RoleATS, urlPrefix: "https://apply.workable.com"},
	{name: "recruitee", label: "Recruitee", kind: kindBoard, role: RoleATS, urlPrefix: "https://recruitee.com"},
	{name: "personio", label: "Personio", kind: kindBoard, role: RoleATS, urlPrefix: "https://personio.de"},
	{name: "wttj", label: "Welcome to the Jungle", kind: kindBoard, role: RoleATS, urlPrefix: "https://app.welcometothejungle.com/companies"},
	{name: "wis", label: "Work in Startups", kind: kindFilter, role: RoleDiscovery, urlPrefix: "https://workinstartups.com", filters: []FilterField{
		{Name: "region", Param: "w", Label: "Region", Options: []FilterOption{
			{Value: "uk", Label: "United Kingdom"},
		}},
	}},
	{name: "linkedin", label: "LinkedIn", kind: kindFilter, role: RoleDiscovery, urlPrefix: "https://www.linkedin.com/jobs", filters: []FilterField{
		{Name: "location", Param: "location", Label: "Location"},
		{Name: "company_id", Param: "f_C", Label: "Company ID", Numeric: true},
		{Name: "recency", Param: "f_TPR", Label: "Recency", Options: []FilterOption{
			{Value: "r86400", Label: "Past 24 hours"},
			{Value: "r604800", Label: "Past week"},
			{Value: "r2592000", Label: "Past month"},
		}},
		{Name: "arrangement", Param: "f_WT", Label: "Work Arrangement", Options: []FilterOption{
			{Value: "1", Label: "On-site"},
			{Value: "2", Label: "Remote"},
			{Value: "3", Label: "Hybrid"},
		}},
		{Name: "experience", Param: "f_E", Label: "Experience Level", Options: []FilterOption{
			{Value: "1", Label: "Internship"},
			{Value: "2", Label: "Entry level"},
			{Value: "3", Label: "Associate"},
			{Value: "4", Label: "Mid-Senior level"},
			{Value: "5", Label: "Director"},
			{Value: "6", Label: "Executive"},
		}},
		{Name: "job_type", Param: "f_JT", Label: "Job Type", Options: []FilterOption{
			{Value: "F", Label: "Full-time"},
			{Value: "P", Label: "Part-time"},
			{Value: "C", Label: "Contract"},
			{Value: "T", Label: "Temporary"},
			{Value: "I", Label: "Internship"},
		}},
		{Name: "geo_id", Param: "geoId", Label: "Geo ID", Numeric: true},
		{Name: "distance", Param: "f_D", Label: "Distance", Options: []FilterOption{
			{Value: "10", Label: "10 miles"},
			{Value: "25", Label: "25 miles"},
			{Value: "50", Label: "50 miles"},
			{Value: "75", Label: "75 miles"},
			{Value: "100", Label: "100 miles"},
		}},
		{Name: "salary_band", Param: "f_SB2", Label: "Salary Band", Options: []FilterOption{
			{Value: "1", Label: "$40,000+"},
			{Value: "2", Label: "$60,000+"},
			{Value: "3", Label: "$80,000+"},
			{Value: "4", Label: "$100,000+"},
			{Value: "5", Label: "$120,000+"},
			{Value: "6", Label: "$140,000+"},
			{Value: "7", Label: "$160,000+"},
			{Value: "8", Label: "$180,000+"},
			{Value: "9", Label: "$200,000+"},
		}},
	}},
	{name: "indeed", label: "Indeed", kind: kindFilter, role: RoleDiscovery, urlPrefix: "https://uk.indeed.com", cardComplete: true, filters: []FilterField{
		{Name: "location", Param: "l", Label: "Location"},
		{Name: "radius", Param: "radius", Label: "Radius", Options: []FilterOption{
			{Value: "5", Label: "5 miles"},
			{Value: "10", Label: "10 miles"},
			{Value: "15", Label: "15 miles"},
			{Value: "25", Label: "25 miles"},
			{Value: "50", Label: "50 miles"},
			{Value: "100", Label: "100 miles"},
		}},
		{Name: "recency", Param: "fromage", Label: "Recency", Options: []FilterOption{
			{Value: "1", Label: "Past 24 hours"},
			{Value: "3", Label: "Past 3 days"},
			{Value: "7", Label: "Past week"},
			{Value: "14", Label: "Past 2 weeks"},
		}},
	}},
	{name: "remoteok", label: "RemoteOK", kind: kindFilter, role: RoleDiscovery, urlPrefix: "https://remoteok.com", cardComplete: true},
	{name: "remotive", label: "Remotive", kind: kindFilter, role: RoleDiscovery, urlPrefix: "https://remotive.com", cardComplete: true},
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
			Kind:      string(e.kind),
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

// ValidFilterValue reports whether value is acceptable for the named filter on the source:
// a declared option for enumerated fields, digits for numeric fields, anything for free-form
// fields. It is false for unknown sources and unknown fields.
func ValidFilterValue(source, field, value string) bool {
	fields, ok := LookupFilterFields(source)
	if !ok {
		return false
	}
	for _, f := range fields {
		if f.Name != field {
			continue
		}
		switch {
		case len(f.Options) > 0:
			return slices.ContainsFunc(f.Options, func(o FilterOption) bool { return o.Value == value })
		case f.Numeric:
			return value != "" && strings.Trim(value, "0123456789") == ""
		}
		return true
	}
	return false
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

// CardComplete reports whether the source's listing card is already the full job.
func CardComplete(name string) bool {
	e, _ := findEntry(name)
	return e.cardComplete
}

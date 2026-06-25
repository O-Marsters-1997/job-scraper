package sources

// sourceKind classifies how a source target's value is interpreted.
type sourceKind int

const (
	// kindBoard identifies ATS sources whose value is a board token (e.g. "acmecorp").
	kindBoard sourceKind = iota
	// kindURL identifies sources whose value is a URL (e.g. a LinkedIn search URL).
	kindURL
	// kindFilter identifies sources whose value is a keyword string and whose structured
	// parameters (e.g. region) are stored in the filters map. The source builds its own
	// start URLs from the user input merged with hardcoded params.
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
	URLPrefix string        `json:"url_prefix"`
	Filters   []FilterField `json:"filters"` // non-empty only for kindFilter sources
}

type registryEntry struct {
	name      string
	label     string
	kind      sourceKind
	urlPrefix string
	filters   []FilterField // non-nil only for kindFilter sources
}

// The names here must match the Name field baked into each source's Config.
var entries = []registryEntry{
	{name: "greenhouse", label: "Greenhouse", kind: kindBoard, urlPrefix: "https://boards.greenhouse.io"},
	{name: "lever", label: "Lever", kind: kindBoard, urlPrefix: "https://jobs.lever.co"},
	{name: "ashby", label: "Ashby", kind: kindBoard, urlPrefix: "https://jobs.ashbyhq.com"},
	{name: "workable", label: "Workable", kind: kindBoard, urlPrefix: "https://apply.workable.com"},
	{name: "recruitee", label: "Recruitee", kind: kindBoard, urlPrefix: "https://recruitee.com"},
	{name: "personio", label: "Personio", kind: kindBoard, urlPrefix: "https://personio.de"},
	{name: "wis", label: "Work in Startups", kind: kindFilter, urlPrefix: "https://workinstartups.com", filters: []FilterField{
		{Name: "region", Label: "Region", Required: false},
	}},
	{name: "linkedin", label: "LinkedIn", kind: kindURL, urlPrefix: "https://www.linkedin.com/jobs"},
	{name: "indeed", label: "Indeed", kind: kindURL, urlPrefix: "https://www.indeed.com"},
}

// Sources returns the full registry as a slice of serialisable SourceInfo values.
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

func IsFilterSource(name string) bool {
	e, ok := findEntry(name)
	return ok && e.kind == kindFilter
}

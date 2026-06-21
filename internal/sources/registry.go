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

// FilterField describes a structured filter parameter accepted by a kindFilter source.
type FilterField struct {
	Name     string // JSON/map key, e.g. "region"
	Label    string // human-readable label, e.g. "Region"
	Required bool
}

type registryEntry struct {
	name      string
	kind      sourceKind
	urlPrefix string
	filters   []FilterField // non-nil only for kindFilter sources
}

// The names here must match the Name field baked into each source's Config.
var entries = []registryEntry{
	{name: "greenhouse", kind: kindBoard, urlPrefix: "https://boards.greenhouse.io"},
	{name: "lever", kind: kindBoard, urlPrefix: "https://jobs.lever.co"},
	{name: "ashby", kind: kindBoard, urlPrefix: "https://jobs.ashbyhq.com"},
	{name: "workable", kind: kindBoard, urlPrefix: "https://apply.workable.com"},
	{name: "recruitee", kind: kindBoard, urlPrefix: "https://recruitee.com"},
	{name: "personio", kind: kindBoard, urlPrefix: "https://personio.de"},
	{name: "wis", kind: kindFilter, urlPrefix: "https://workinstartups.com", filters: []FilterField{
		{Name: "region", Label: "Region", Required: false},
	}},
	{name: "linkedin", kind: kindURL, urlPrefix: "https://www.linkedin.com/jobs"},
	{name: "indeed", kind: kindURL, urlPrefix: "https://www.indeed.com"},
}

// SupportedSources returns the names of all supported sources.
func SupportedSources() []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.name
	}
	return names
}

// LookupSource returns the URL prefix and whether the source is URL-based (kindURL), plus
// whether the source name is known at all. kindFilter sources return isURL=false.
func LookupSource(name string) (urlPrefix string, isURL bool, ok bool) {
	for _, e := range entries {
		if e.name == name {
			return e.urlPrefix, e.kind == kindURL, true
		}
	}
	return "", false, false
}

// LookupFilterFields returns the declared filter fields for a kindFilter source,
// plus whether the source is a filter source at all.
func LookupFilterFields(name string) ([]FilterField, bool) {
	for _, e := range entries {
		if e.name == name && e.kind == kindFilter {
			return e.filters, true
		}
	}
	return nil, false
}

// IsFilterSource reports whether name is a kindFilter source.
func IsFilterSource(name string) bool {
	for _, e := range entries {
		if e.name == name {
			return e.kind == kindFilter
		}
	}
	return false
}

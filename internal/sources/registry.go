package sources

// sourceKind classifies how a source target's value is interpreted.
type sourceKind int

const (
	// kindBoard identifies ATS sources whose value is a board token (e.g. "acmecorp").
	kindBoard sourceKind = iota
	// kindURL identifies sources whose value is a URL (e.g. a LinkedIn search URL).
	kindURL
)

type registryEntry struct {
	name      string
	kind      sourceKind
	urlPrefix string
}

// The names here must match the Name field baked into each source's Config.
var entries = []registryEntry{
	{name: "greenhouse", kind: kindBoard, urlPrefix: "https://boards.greenhouse.io"},
	{name: "lever", kind: kindBoard, urlPrefix: "https://jobs.lever.co"},
	{name: "ashby", kind: kindBoard, urlPrefix: "https://jobs.ashbyhq.com"},
	{name: "workable", kind: kindBoard, urlPrefix: "https://apply.workable.com"},
	{name: "recruitee", kind: kindBoard, urlPrefix: "https://recruitee.com"},
	{name: "personio", kind: kindBoard, urlPrefix: "https://personio.de"},
	{name: "wis", kind: kindURL, urlPrefix: "https://workinstartups.com"},
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

// LookupSource returns the URL prefix and whether the source is URL-based, plus
// whether the source name is known at all.
func LookupSource(name string) (urlPrefix string, isURL bool, ok bool) {
	for _, e := range entries {
		if e.name == name {
			return e.urlPrefix, e.kind == kindURL, true
		}
	}
	return "", false, false
}

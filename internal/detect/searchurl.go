package detect

import (
	neturl "net/url"
	"slices"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/sourcespec"
)

// SearchURL is a board search page reduced to the fields the source adapters
// accept. Value is the keyword query, or for Indeed the normalised URL.
type SearchURL struct {
	Source  string
	Value   string
	Filters map[string]string
	Dropped []string
}

type paramSpec struct {
	param  string
	filter string
}

type boardSpec struct {
	source        string
	hosts         []string
	paths         []string
	canonical     string
	keywordsParam string
	params        []paramSpec
	dropped       []string
	silent        []string
}

var searchBoards = []boardSpec{
	{
		source:        "linkedin",
		hosts:         []string{"linkedin.com"},
		paths:         []string{"/jobs/search", "/jobs/search-results"},
		canonical:     "https://www.linkedin.com/jobs/search/",
		keywordsParam: "keywords",
		dropped:       []string{"currentJobId", "origin", "referralSearchId", "refresh", "position", "pageNum", "trk", "trackingId"},
		silent:        []string{"start"},
	},
	{
		source:        "wis",
		hosts:         []string{"workinstartups.com"},
		paths:         []string{"/search"},
		canonical:     "https://workinstartups.com/search",
		keywordsParam: "q",
		silent:        []string{"p", "per_page"},
	},
	{
		source:        "indeed",
		hosts:         []string{"indeed.com", "indeed.co.uk"},
		paths:         []string{"/jobs"},
		keywordsParam: "q",
		params: []paramSpec{
			{"l", "l"},
			{"fromage", "fromage"},
			{"radius", "radius"},
			{"jt", "jt"},
			{"sort", "sort"},
		},
		dropped: []string{"vjk", "from", "advn", "vjs", "sc"},
		silent:  []string{"start"},
	},
}

var unsupportedBoards = []struct {
	label string
	hosts []string
}{
	{"RemoteOK", []string{"remoteok.com", "remoteok.io"}},
	{"Remotive", []string{"remotive.com", "remotive.io"}},
}

func parseHTTP(raw string) (*neturl.URL, bool) {
	u, err := neturl.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, false
	}
	return u, true
}

func hostMatches(host, want string) bool {
	host = strings.ToLower(host)
	return host == want || strings.HasSuffix(host, "."+want)
}

func (b boardSpec) matchesHost(host string) bool {
	return slices.ContainsFunc(b.hosts, func(h string) bool { return hostMatches(host, h) })
}

func (b boardSpec) matchesPath(path string) bool {
	return slices.Contains(b.paths, strings.TrimSuffix(path, "/"))
}

func (b boardSpec) paramSpecs() []paramSpec {
	fields, ok := sourcespec.LookupFilterFields(b.source)
	if !ok {
		return b.params
	}
	specs := make([]paramSpec, len(fields))
	for i, f := range fields {
		specs[i] = paramSpec{param: f.Param, filter: f.Name}
	}
	return specs
}

func (b boardSpec) filterName(param string) (string, bool) {
	for _, p := range b.paramSpecs() {
		if p.param == param {
			return p.filter, true
		}
	}
	return "", false
}

func validFilterValue(source, filter, value string) bool {
	if !sourcespec.IsFilterSource(source) {
		return true
	}
	return sourcespec.ValidFilterValue(source, filter, value)
}

// UnsupportedBoard reports the label of a recognised board whose search URLs
// cannot be parsed yet.
func UnsupportedBoard(raw string) (label string, ok bool) {
	u, valid := parseHTTP(raw)
	if !valid {
		return "", false
	}
	for _, b := range unsupportedBoards {
		if slices.ContainsFunc(b.hosts, func(h string) bool { return hostMatches(u.Hostname(), h) }) {
			return b.label, true
		}
	}
	return "", false
}

// ParseSearchURL reduces a pasted LinkedIn, Indeed or Work in Startups search
// URL to its source, keyword query and named filters. Params the adapter would
// not use are reported in Dropped; paging params are discarded silently.
func ParseSearchURL(raw string) (SearchURL, bool) {
	u, ok := parseHTTP(raw)
	if !ok {
		return SearchURL{}, false
	}
	for _, b := range searchBoards {
		if !b.matchesHost(u.Hostname()) || !b.matchesPath(u.Path) {
			continue
		}
		return b.parse(u), true
	}
	return SearchURL{}, false
}

func (b boardSpec) parse(u *neturl.URL) SearchURL {
	query := u.Query()
	out := SearchURL{Source: b.source, Filters: map[string]string{}, Dropped: []string{}}
	for param, values := range query {
		value := values[0]
		switch {
		case param == b.keywordsParam, slices.Contains(b.silent, param), value == "":
		case slices.Contains(b.dropped, param), strings.HasPrefix(param, "utm_"):
			out.Dropped = append(out.Dropped, param)
		default:
			filter, known := b.filterName(param)
			if !known || !validFilterValue(b.source, filter, value) {
				out.Dropped = append(out.Dropped, param)
				continue
			}
			out.Filters[filter] = value
		}
	}
	slices.Sort(out.Dropped)

	keywords := query.Get(b.keywordsParam)
	if b.source != "indeed" {
		out.Value = keywords
		return out
	}
	out.Value = buildIndeed(strings.ToLower(u.Hostname()), keywords, out.Filters)
	out.Filters = map[string]string{}
	return out
}

func buildIndeed(host, keywords string, params map[string]string) string {
	v := neturl.Values{}
	if keywords != "" {
		v.Set("q", keywords)
	}
	for k, val := range params {
		v.Set(k, val)
	}
	u := neturl.URL{Scheme: "https", Host: host, Path: "/jobs", RawQuery: v.Encode()}
	return u.String()
}

// BuildSearchURL returns the board page for a search, or "" for a source with
// no search page. It inverts ParseSearchURL.
func BuildSearchURL(source, value string, filters map[string]string) string {
	for _, b := range searchBoards {
		if b.source != source {
			continue
		}
		if source == "indeed" {
			parsed, ok := ParseSearchURL(value)
			if !ok {
				return ""
			}
			return parsed.Value
		}
		v := neturl.Values{}
		if value != "" {
			v.Set(b.keywordsParam, value)
		}
		for _, p := range b.paramSpecs() {
			if f := filters[p.filter]; f != "" {
				v.Set(p.param, f)
			}
		}
		if len(v) == 0 {
			return b.canonical
		}
		return b.canonical + "?" + v.Encode()
	}
	return ""
}

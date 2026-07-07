package detect

import (
	"log/slog"
	"regexp"
)

// sniffPattern recognises one embedded-ATS marker in a raw page body and
// extracts a source+token from the regex match. Table-driven per ADR 0004.
type sniffPattern struct {
	name    string
	pattern *regexp.Regexp
	extract func(matches []string) (source, token string)
}

var sniffPatterns = []sniffPattern{
	{
		// ResolveBoard workaround: for this embed shape, the path segment
		// after /embed/ is always "job_board", never the real token — the
		// token lives in the for= query param instead. Handled here rather
		// than in ResolveBoard, which must stay a pure URL parser.
		name:    "greenhouse-embed",
		pattern: regexp.MustCompile(`boards\.greenhouse\.io/embed/job_board\?[^\s"'<>]*for=([a-zA-Z0-9_-]+)`),
		extract: func(m []string) (string, string) { return "greenhouse", m[1] },
	},
	{
		name:    "lever",
		pattern: regexp.MustCompile(`jobs\.lever\.co/([a-zA-Z0-9_-]+)`),
		extract: func(m []string) (string, string) { return "lever", m[1] },
	},
	{
		name:    "ashby",
		pattern: regexp.MustCompile(`jobs\.ashbyhq\.com/([a-zA-Z0-9_-]+)`),
		extract: func(m []string) (string, string) { return "ashby", m[1] },
	},
}

// workdayPattern is detect-only: Workday is not a supported ATS in this
// registry, so a match is logged rather than turned into a source/token.
var workdayPattern = regexp.MustCompile(`[a-zA-Z0-9-]+\.myworkdayjobs\.com`)

// SniffATS scans a raw page body for embedded ATS board markers that don't
// surface as plain <a href> links — inline <script> config blobs, SPA
// hydration state, and similar. It is the fallback extractor for pages where
// a link scan finds nothing.
func SniffATS(body []byte) (source, token string, ok bool) {
	for _, p := range sniffPatterns {
		m := p.pattern.FindSubmatch(body)
		if m == nil {
			continue
		}
		strs := make([]string, len(m))
		for i, b := range m {
			strs[i] = string(b)
		}
		source, token = p.extract(strs)
		return source, token, true
	}

	if m := workdayPattern.Find(body); m != nil {
		slog.Info("detect: workday board detected, not a supported ATS", slog.String("match", string(m)))
	}

	return "", "", false
}

package detect

import (
	"log/slog"
	"regexp"
)

type sniffPattern struct {
	name    string
	pattern *regexp.Regexp
	extract func(matches []string) (source, token string)
}

var sniffPatterns = []sniffPattern{
	{
		// Greenhouse always puts the literal "job_board" in this embed's path
		// segment; the real token lives in the for= query param instead.
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

var workdayPattern = regexp.MustCompile(`[a-zA-Z0-9-]+\.myworkdayjobs\.com`)

// SniffATS scans a raw page body for embedded ATS board markers that a link scan missed.
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

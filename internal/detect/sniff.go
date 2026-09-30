package detect

import (
	"log/slog"
	"regexp"
)

type sniffPattern struct {
	source  string
	pattern *regexp.Regexp
}

// Greenhouse always puts the literal "job_board" in the embed's path segment;
// the real token lives in the for= query param instead.
var sniffPatterns = []sniffPattern{
	{"greenhouse", regexp.MustCompile(`boards\.greenhouse\.io/embed/job_board\?[^\s"'<>]*for=([a-zA-Z0-9_-]+)`)},
	{"lever", regexp.MustCompile(`jobs\.lever\.co/([a-zA-Z0-9_-]+)`)},
	{"ashby", regexp.MustCompile(`jobs\.ashbyhq\.com/([a-zA-Z0-9_-]+)`)},
}

var workdayPattern = regexp.MustCompile(`[a-zA-Z0-9-]+\.myworkdayjobs\.com`)

// SniffATS scans a raw page body for embedded ATS board markers that a link scan missed.
func SniffATS(body []byte) (source, token string, ok bool) {
	for _, p := range sniffPatterns {
		m := p.pattern.FindSubmatch(body)
		if m == nil {
			continue
		}
		return p.source, string(m[1]), true
	}

	if m := workdayPattern.Find(body); m != nil {
		slog.Info("detect: workday board detected, not a supported ATS", slog.String("match", string(m)))
	}

	return "", "", false
}

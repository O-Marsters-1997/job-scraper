package detect

import (
	"log/slog"
	"regexp"
	"strings"
)

type sniffPattern struct {
	source  string
	pattern *regexp.Regexp
}

// Greenhouse always puts the literal "job_board" in the embed's path segment;
// the real token lives in the for= query param instead.
var sniffPatterns = []sniffPattern{
	{"greenhouse", regexp.MustCompile(`boards\.greenhouse\.io/embed/job_board(?:/js)?\?[^\s"'<>]*for=([a-zA-Z0-9_-]+)`)},
	{"lever", regexp.MustCompile(`jobs\.lever\.co/([a-zA-Z0-9_-]+)`)},
	{"ashby", regexp.MustCompile(`(?:jobs\.ashbyhq\.com|api\.ashbyhq\.com/posting-api/job-board)/([a-zA-Z0-9_-]+)`)},
	{"workable", regexp.MustCompile(`apply\.workable\.com/([a-zA-Z0-9_-]+)`)},
	{"recruitee", regexp.MustCompile(`([a-zA-Z0-9-]+)\.recruitee\.com`)},
	{"personio", regexp.MustCompile(`([a-zA-Z0-9-]+)\.jobs\.personio\.(?:de|com)`)},
	{"pinpoint", regexp.MustCompile(`([a-zA-Z0-9-]+)\.pinpointhq\.com`)},
	{"teamtailor", regexp.MustCompile(`([a-zA-Z0-9-]+)\.teamtailor\.com`)},
	{"hibob", regexp.MustCompile(`([a-zA-Z0-9-]+)\.careers\.hibob\.com`)},
	{"smartrecruiters", regexp.MustCompile(`(?:(?:jobs|careers)\.smartrecruiters\.com/(?:oneclick-ui/company/)?|widget\.smartrecruiters\.com/)([a-zA-Z0-9_-]+)`)},
}

func reservedSubdomain(label string) bool {
	switch strings.ToLower(label) {
	case "www", "jobs", "api", "app", "career", "careers", "support":
		return true
	}
	return false
}

var workdayPattern = regexp.MustCompile(`[a-zA-Z0-9-]+\.myworkdayjobs\.com`)

// SniffATS scans a raw page body for embedded ATS board markers that a link scan missed.
func SniffATS(body []byte) (source, token string, ok bool) {
	for _, p := range sniffPatterns {
		for _, m := range p.pattern.FindAllSubmatch(body, -1) {
			if reservedSubdomain(string(m[1])) {
				continue
			}
			return p.source, string(m[1]), true
		}
	}

	if m := workdayPattern.Find(body); m != nil {
		slog.Info("detect: workday board detected, not a supported ATS", slog.String("match", string(m)))
	}

	return "", "", false
}

package detect

import (
	neturl "net/url"
	"strings"
)

type ATSType int

const (
	UnknownHTML ATSType = iota
	Greenhouse
	Lever
	Ashby
	Workable
	Recruitee
	Personio
	Aggregator
)

func Detect(rawURL string) ATSType {
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return UnknownHTML
	}
	host := strings.ToLower(u.Host)
	switch {
	case strings.Contains(host, "greenhouse.io"):
		return Greenhouse
	case host == "jobs.lever.co":
		return Lever
	case host == "jobs.ashbyhq.com":
		return Ashby
	case strings.Contains(host, "workable.com"):
		return Workable
	case strings.Contains(host, "recruitee.com"):
		return Recruitee
	case strings.Contains(host, "personio.de") || strings.Contains(host, "personio.com"):
		return Personio
	case strings.Contains(host, "linkedin.com") || strings.Contains(host, "indeed.com"):
		return Aggregator
	default:
		return UnknownHTML
	}
}

var atsSourceName = map[ATSType]string{
	Greenhouse: "greenhouse",
	Lever:      "lever",
	Ashby:      "ashby",
	Workable:   "workable",
	Recruitee:  "recruitee",
	Personio:   "personio",
}

// ResolveBoard extracts the source name and board token from a direct ATS board
// URL (e.g. https://boards.greenhouse.io/acmecorp -> "greenhouse", "acmecorp").
func ResolveBoard(rawURL string) (source, token string, ok bool) {
	t := Detect(rawURL)
	source, known := atsSourceName[t]
	if !known {
		return "", "", false
	}
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return "", "", false
	}

	//exhaustive:ignore — default handles all path-based ATSes; only subdomain ones are special-cased
	switch t {
	case Recruitee, Personio:
		// Token is the leading host label: {token}.recruitee.com / {token}.[jobs.]personio.de
		labels := strings.Split(strings.ToLower(u.Host), ".")
		if len(labels) < 3 || labels[0] == "www" || labels[0] == "jobs" {
			return "", "", false
		}
		return source, labels[0], true
	default:
		for seg := range strings.SplitSeq(u.Path, "/") {
			if seg != "" && seg != "embed" {
				return source, seg, true
			}
		}
		return "", "", false
	}
}

// RewriteToATS attempts to extract the underlying ATS URL from an aggregator URL.
func RewriteToATS(rawURL string) (string, ATSType, bool) {
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return "", UnknownHTML, false
	}
	host := strings.ToLower(u.Host)

	if strings.Contains(host, "linkedin.com") {
		if ext := u.Query().Get("externalUrl"); ext != "" {
			t := Detect(ext)
			if t != Aggregator && t != UnknownHTML {
				return ext, t, true
			}
		}
	}

	if strings.Contains(host, "indeed.com") {
		if dest := u.Query().Get("url"); dest != "" {
			if unescaped, err := neturl.QueryUnescape(dest); err == nil {
				t := Detect(unescaped)
				if t != Aggregator && t != UnknownHTML {
					return unescaped, t, true
				}
			}
		}
	}

	return "", UnknownHTML, false
}

// BoardURL is the public board page for a source and token, the inverse of
// ResolveBoard. It returns "" for a source that has no board page.
func BoardURL(source, token string) string {
	switch source {
	case "greenhouse":
		return "https://boards.greenhouse.io/" + token
	case "lever":
		return "https://jobs.lever.co/" + token
	case "ashby":
		return "https://jobs.ashbyhq.com/" + token
	case "workable":
		return "https://apply.workable.com/" + token
	case "recruitee":
		return "https://" + token + ".recruitee.com"
	case "personio":
		return "https://" + token + ".jobs.personio.de"
	default:
		return ""
	}
}

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

// RewriteToATS attempts to extract the underlying ATS URL from an aggregator URL.
// Returns the ATS URL, its type, and true on success; ("", UnknownHTML, false) otherwise.
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

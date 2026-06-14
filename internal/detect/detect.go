package detect

import (
	"net/url"
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

// Detect classifies rawURL to its ATSType by hostname/path patterns.
func Detect(rawURL string) ATSType {
	u, err := url.Parse(rawURL)
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
// Stub for Phase 7 — always returns false.
func RewriteToATS(rawURL string) (string, ATSType, bool) {
	return "", UnknownHTML, false
}

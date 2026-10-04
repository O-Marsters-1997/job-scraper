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
	Pinpoint
	Teamtailor
	HiBob
	SmartRecruiters
	Aggregator
)

func Detect(rawURL string) ATSType {
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return UnknownHTML
	}
	host := strings.ToLower(u.Hostname())
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
	case strings.HasSuffix(host, ".pinpointhq.com"):
		return Pinpoint
	case strings.HasSuffix(host, ".teamtailor.com"):
		return Teamtailor
	case strings.HasSuffix(host, ".careers.hibob.com"):
		return HiBob
	case host == "jobs.smartrecruiters.com" || host == "careers.smartrecruiters.com":
		return SmartRecruiters
	case strings.Contains(host, "linkedin.com") || strings.Contains(host, "indeed.com"):
		return Aggregator
	default:
		return UnknownHTML
	}
}

var atsSourceName = map[ATSType]string{
	Greenhouse:      "greenhouse",
	Lever:           "lever",
	Ashby:           "ashby",
	Workable:        "workable",
	Recruitee:       "recruitee",
	Personio:        "personio",
	Pinpoint:        "pinpoint",
	Teamtailor:      "teamtailor",
	HiBob:           "hibob",
	SmartRecruiters: "smartrecruiters",
}

var (
	teamtailorReserved = map[string]bool{"app": true, "career": true, "api": true}
	hibobReserved      = map[string]bool{"app": true, "api": true}
)

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
	case Recruitee, Personio, Pinpoint, Teamtailor, HiBob:
		// Token is the leading host label: {token}.recruitee.com / {token}.[jobs.]personio.de / {token}.pinpointhq.com / {token}.teamtailor.com / {token}.careers.hibob.com
		labels := strings.Split(strings.ToLower(u.Host), ".")
		reserved := labels[0] == "www" || labels[0] == "jobs" || (t == Teamtailor && teamtailorReserved[labels[0]]) || (t == HiBob && hibobReserved[labels[0]])
		if len(labels) < 3 || reserved {
			return "", "", false
		}
		return source, labels[0], true
	case SmartRecruiters:
		segs := strings.Split(strings.Trim(u.Path, "/"), "/")
		if segs[0] == "oneclick-ui" {
			if len(segs) >= 3 && segs[1] == "company" && segs[2] != "" {
				return source, segs[2], true
			}
			return "", "", false
		}
		if segs[0] == "" {
			return "", "", false
		}
		return source, segs[0], true
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
	host := strings.ToLower(u.Hostname())

	var dest string
	switch {
	case strings.Contains(host, "linkedin.com"):
		dest = u.Query().Get("externalUrl")
	case strings.Contains(host, "indeed.com"):
		dest, _ = neturl.QueryUnescape(u.Query().Get("url"))
	}
	if t := Detect(dest); dest != "" && t != Aggregator && t != UnknownHTML {
		return dest, t, true
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
	case "pinpoint":
		return "https://" + token + ".pinpointhq.com"
	case "teamtailor":
		return "https://" + token + ".teamtailor.com"
	case "hibob":
		return "https://" + token + ".careers.hibob.com"
	case "smartrecruiters":
		return "https://jobs.smartrecruiters.com/" + token
	default:
		return ""
	}
}

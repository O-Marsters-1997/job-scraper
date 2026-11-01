package detect

import (
	neturl "net/url"
	"strings"
)

// ResolveListing maps a LinkedIn or Work in Startups job URL to its source and the canonical URL
// that source's cards carry; ok is false for any other URL.
func ResolveListing(rawURL string) (source, canonicalURL string, ok bool) {
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return "", "", false
	}
	host := strings.ToLower(u.Hostname())
	segs := strings.Split(strings.Trim(u.Path, "/"), "/")
	switch {
	case host == "linkedin.com" || strings.HasSuffix(host, ".linkedin.com"):
		id := u.Query().Get("currentJobId")
		if len(segs) >= 3 && segs[0] == "jobs" && segs[1] == "view" {
			id = segs[2][strings.LastIndex(segs[2], "-")+1:]
		}
		if !isDigits(id) {
			return "", "", false
		}
		return "linkedin", "https://www.linkedin.com/jobs/view/" + id, true
	case host == "workinstartups.com" || host == "www.workinstartups.com":
		if len(segs) == 2 && segs[0] == "details" && isDigits(segs[1]) {
			return "wis", "https://workinstartups.com/details/" + segs[1], true
		}
	}
	return "", "", false
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

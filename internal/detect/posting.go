package detect

import (
	neturl "net/url"
	"strconv"
	"strings"
)

// ResolvePosting extracts source, board token and posting ID from a Greenhouse, Lever or Ashby
// job URL; ok is false for any other URL, including other ATS hosts.
func ResolvePosting(rawURL string) (source, token, postingID string, ok bool) {
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return "", "", "", false
	}
	segs := strings.Split(strings.Trim(u.Path, "/"), "/")
	switch Detect(rawURL) {
	case Greenhouse:
		token, postingID, ok = greenhousePosting(u, segs)
		return "greenhouse", token, postingID, ok
	case Lever:
		token, postingID, ok = tokenAndID(segs)
		return "lever", token, postingID, ok
	case Ashby:
		token, postingID, ok = tokenAndID(segs)
		return "ashby", token, postingID, ok
	case UnknownHTML, Workable, Recruitee, Personio, Pinpoint, Teamtailor, HiBob, SmartRecruiters, Aggregator:
		return "", "", "", false
	default:
		return "", "", "", false
	}
}

func greenhousePosting(u *neturl.URL, segs []string) (token, postingID string, ok bool) {
	q := u.Query()
	switch {
	case len(segs) >= 3 && segs[1] == "jobs" && segs[0] != "embed":
		token, postingID = segs[0], segs[2]
	case len(segs) == 2 && segs[0] == "embed" && segs[1] == "job_app":
		token, postingID = q.Get("for"), q.Get("token")
	case len(segs) == 1 && segs[0] != "" && segs[0] != "embed":
		token, postingID = segs[0], q.Get("gh_jid")
	}
	id, err := strconv.ParseUint(postingID, 10, 64)
	if token == "" || err != nil {
		return "", "", false
	}
	return token, strconv.FormatUint(id, 10), true
}

func tokenAndID(segs []string) (token, postingID string, ok bool) {
	if len(segs) < 2 || segs[0] == "" || segs[1] == "" {
		return "", "", false
	}
	return segs[0], segs[1], true
}

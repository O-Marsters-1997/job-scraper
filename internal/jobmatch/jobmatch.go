package jobmatch

import (
	"regexp"
	"slices"
	"strings"
)

const remote = "remote"

var (
	genderTag       = regexp.MustCompile(`[(\[]\s*(?:all genders|[mfwdx*](?:\s*/\s*[mfwdx*])+)\s*[)\]]`)
	trailingSegment = regexp.MustCompile(`^(.*\S)\s+[-–—|]\s+([^-–—|]+)$`)
	trailingBracket = regexp.MustCompile(`^(.*\S)\s*[(\[]([^()\[\]]*)[)\]]$`)
	bracketed       = regexp.MustCompile(`[(\[][^()\[\]]*[)\]]`)
	nonAlnum        = regexp.MustCompile(`[^\p{L}\p{N}]+`)
)

var placeWords = map[string]bool{
	"remote": true, "hybrid": true, "onsite": true, "on": true, "site": true, "office": true,
	"first": true, "only": true, "based": true, "fully": true, "flexible": true,
	"anywhere": true, "worldwide": true, "global": true,
	"uk": true, "gb": true, "united": true, "kingdom": true, "england": true, "scotland": true,
	"wales": true, "ireland": true, "europe": true, "emea": true, "eu": true, "us": true, "usa": true,
	"london": true, "manchester": true, "edinburgh": true, "glasgow": true, "bristol": true,
	"cambridge": true, "oxford": true, "leeds": true, "birmingham": true, "belfast": true,
}

var remoteWords = []string{"remote", "anywhere", "worldwide"}

func Title(raw string) string {
	s := strings.TrimSpace(genderTag.ReplaceAllString(strings.ToLower(raw), " "))
	for {
		head, ok := stripPlaceSuffix(s)
		if !ok {
			return collapse(s)
		}
		s = head
	}
}

func Location(raw string) string {
	s := strings.ToLower(raw)
	if mentionsRemote(s) {
		return remote
	}
	city, _, _ := strings.Cut(s, ",")
	return collapse(bracketed.ReplaceAllString(city, " "))
}

func LocationsCompatible(a, b string) bool {
	return a == b || a == "" || b == "" || a == remote || b == remote
}

func stripPlaceSuffix(s string) (string, bool) {
	for _, re := range []*regexp.Regexp{trailingSegment, trailingBracket} {
		if m := re.FindStringSubmatch(s); m != nil && onlyPlaceWords(m[2]) {
			return strings.TrimSpace(m[1]), true
		}
	}
	return s, false
}

func mentionsRemote(s string) bool {
	return slices.ContainsFunc(strings.Fields(collapse(s)), func(w string) bool {
		return slices.Contains(remoteWords, w)
	})
}

func onlyPlaceWords(s string) bool {
	words := strings.Fields(collapse(s))
	for _, w := range words {
		if !placeWords[w] {
			return false
		}
	}
	return len(words) > 0
}

func collapse(s string) string {
	return strings.Join(strings.Fields(nonAlnum.ReplaceAllString(s, " ")), " ")
}

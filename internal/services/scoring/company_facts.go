package scoring

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

var (
	sizeOptionIDs  = []string{"size:startup", "size:scaleup", "size:large", "size:enterprise"}
	stageOptionIDs = []string{"stage:seed", "stage:series_a", "stage:series_b", "stage:series_c_plus"}
	headcountRange = regexp.MustCompile(`^\s*(\d[\d,]*)(?:\s*[-–]\s*(\d[\d,]*))?\s*(\+)?`)
)

// companyFactAnswers answers the size and stage Options from a company
// profile, keyed by Option ID. An Option the profile cannot place is omitted,
// and stage:public is always omitted: funding rounds don't say who has listed.
func companyFactAnswers(p dto.CompanyProfile) map[string]dto.Answer {
	out := make(map[string]dto.Answer)
	addExclusive(out, sizeOptionIDs, sizeIndex(p.Size))
	addExclusive(out, stageOptionIDs, stageIndex(p.FundingRounds))
	return out
}

func addExclusive(out map[string]dto.Answer, ids []string, yes int) {
	if yes < 0 {
		return
	}
	for i, id := range ids {
		if i == yes {
			out[id] = dto.Answer{PYes: 1}
		} else {
			out[id] = dto.Answer{PNo: 1}
		}
	}
}

// sizeIndex buckets a headcount range such as "201-500" or "5000+ employees" by
// its upper bound, returning -1 when the text does not start with a number.
func sizeIndex(size string) int {
	m := headcountRange.FindStringSubmatch(size)
	if m == nil {
		return -1
	}
	upper := m[1]
	if m[2] != "" {
		upper = m[2]
	}
	n, err := strconv.Atoi(strings.ReplaceAll(upper, ",", ""))
	if err != nil {
		return -1
	}
	if m[3] != "" {
		n++
	}
	switch {
	case n <= 50:
		return 0
	case n <= 500:
		return 1
	case n <= 5000:
		return 2
	default:
		return 3
	}
}

func stageIndex(fundingRounds int) int {
	if fundingRounds <= 0 {
		return -1
	}
	return min(fundingRounds, len(stageOptionIDs)) - 1
}

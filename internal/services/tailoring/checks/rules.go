package checks

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	checkBanned = "banned_words"
	checkLength = "slot_length"
	checkPages  = "page_count"

	maxLengthPercent = 115
)

var bannedRe = regexp.MustCompile(`(?i)\b(leverag(?:e|es|ed|ing)|spearhead(?:s|ed|ing)?|synerg(?:y|ies)|passionate|results-driven|cutting-edge|utili[sz](?:e|es|ed|ing)|delv(?:e|es|ed|ing)|seamless(?:ly)?|game-chang(?:er|ing)|best-in-class|rockstar|ninja|go-getter)\b`)

func BannedWords(d Draft) []Finding {
	var out []Finding
	for _, s := range allSlots(d) {
		seen := map[string]bool{}
		for _, w := range bannedRe.FindAllString(s.Text, -1) {
			key := strings.ToLower(w)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, Finding{
				Check: checkBanned, Severity: Block, SlotID: s.ID,
				Message: fmt.Sprintf("banned word %q", w),
			})
		}
	}
	return out
}

// SlotLength blocks any slot longer than 1.15 times its original text.
func SlotLength(d Draft) []Finding {
	var out []Finding
	for _, s := range allSlots(d) {
		base := utf8.RuneCountInString(s.BaseText)
		got := utf8.RuneCountInString(s.Text)
		if base == 0 || got*100 <= base*maxLengthPercent {
			continue
		}
		out = append(out, Finding{
			Check: checkLength, Severity: Block, SlotID: s.ID,
			Message: fmt.Sprintf("%d characters is %.2f times the original %d; the limit is %.2f",
				got, float64(got)/float64(base), base, float64(maxLengthPercent)/100),
		})
	}
	return out
}

// PageCount blocks a Draft that runs onto more pages than the base CV.
func PageCount(d Draft) []Finding {
	if d.DraftPages == 0 || d.DraftPages <= d.BasePages {
		return nil
	}
	return []Finding{{
		Check: checkPages, Severity: Block,
		Message: fmt.Sprintf("the Draft is %d pages; the base CV is %d", d.DraftPages, d.BasePages),
	}}
}

func allSlots(d Draft) []Slot {
	var out []Slot
	for _, p := range d.Positions {
		out = append(out, p.Bullets...)
	}
	if d.Profile != nil {
		out = append(out, *d.Profile)
	}
	return out
}

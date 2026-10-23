package cvedit

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func TailorViolation(original, result string, maxChars int, maxShare float64) string {
	orig, got := words(original), words(result)
	switch {
	case len(got) > len(orig):
		return "the tailored line has more words than the original"
	case maxChars > 0 && utf8.RuneCountInString(result) > maxChars:
		return "the tailored line is longer than the original"
	case len(orig) > 0 && float64(len(orig)-commonWords(orig, got))/float64(len(orig)) > maxShare:
		return "the tailored line changes too much of the original"
	}
	return ""
}

func words(s string) []string {
	fields := strings.Fields(strings.ToLower(s))
	for i, f := range fields {
		fields[i] = strings.TrimFunc(f, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	}
	return fields
}

func commonWords(a, b []string) int {
	prev := make([]int, len(b)+1)
	for _, x := range a {
		cur := make([]int, len(b)+1)
		for j, y := range b {
			if x == y {
				cur[j+1] = prev[j] + 1
			} else {
				cur[j+1] = max(prev[j+1], cur[j])
			}
		}
		prev = cur
	}
	return prev[len(b)]
}

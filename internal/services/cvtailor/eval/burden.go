package eval

import (
	"strings"
	"unicode"
)

func EditBurden(output, final string) float64 {
	a, b := words(output), words(final)
	if len(b) == 0 {
		if len(a) == 0 {
			return 0
		}
		return 1
	}
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i, x := range a {
		cur := make([]int, len(b)+1)
		cur[0] = i + 1
		for j, y := range b {
			cost := 1
			if x == y {
				cost = 0
			}
			cur[j+1] = min(prev[j+1]+1, cur[j]+1, prev[j]+cost)
		}
		prev = cur
	}
	return float64(prev[len(b)]) / float64(len(b))
}

func words(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

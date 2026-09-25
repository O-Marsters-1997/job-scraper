package slug

import (
	"strings"
	"unicode"
)

// Make lowercases s, maps each space or hyphen to a hyphen, drops any other
// non-alphanumeric rune, and trims leading/trailing hyphens, e.g. "Acme Corp" ->
// "acme-corp".
func Make(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case unicode.IsSpace(r) || r == '-':
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

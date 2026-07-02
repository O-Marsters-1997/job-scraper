package sources

import (
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
)

var (
	// salaryRe matches a currency amount or range in plain text, e.g.
	// "£80,000 to £95,000", "$120,000", "€70,000 – €90,000", "£90,000.00 to £140,000.00".
	// Intentionally not matching "£80k"-style; refine if real snapshots show it.
	salaryRe = regexp.MustCompile(`[£$€]\s?\d{1,3}(?:,\d{3})+(?:\.\d+)?(?:\s*(?:-|–|to)\s*[£$€]?\s?\d{1,3}(?:,\d{3})+(?:\.\d+)?)?`)

	remoteRe = regexp.MustCompile(`(?i)\bremote\b|\bwork from home\b|\bwfh\b`)
	hybridRe = regexp.MustCompile(`(?i)\bhybrid\b`)
	onsiteRe = regexp.MustCompile(`(?i)\bon[\s-]?site\b|\bin[\s-]?office\b|\boffice[\s-]?based\b`)
)

// ParseHTML wraps goquery.NewDocumentFromReader with a consistent error. Callers keep
// their own Length()/empty guards on the returned document.
func ParseHTML(r io.Reader) (*goquery.Document, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}
	return doc, nil
}

// ParseSalaryRaw extracts a raw salary string from text using a currency regex.
// Returns an empty string when no salary is found.
func ParseSalaryRaw(text string) string {
	return strings.TrimSpace(salaryRe.FindString(text))
}

// DetectWorkArrangement classifies free text as remote/hybrid/onsite, or "" when no
// signal is present. Sources with an authoritative structured signal (e.g. a badge)
// should check that first and use this only as a text fallback.
func DetectWorkArrangement(text string) string {
	switch {
	case remoteRe.MatchString(text):
		return "remote"
	case hybridRe.MatchString(text):
		return "hybrid"
	case onsiteRe.MatchString(text):
		return "onsite"
	default:
		return ""
	}
}

// WarnDefaulted logs that a required field was defaulted because it could not be parsed
// from the page.
func WarnDefaulted(source, field, url string) {
	slog.Warn("defaulted field",
		slog.String("source", source),
		slog.String("field", field),
		slog.String("url", url),
	)
}

// Slugify lowercases s, maps each space or hyphen to a hyphen, drops any other
// non-alphanumeric rune, and trims leading/trailing hyphens, e.g. "Acme Corp" ->
// "acme-corp".
func Slugify(s string) string {
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

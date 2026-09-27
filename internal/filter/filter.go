package filter

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/slug"
)

// Reject reports whether job trips a company or blocked-tech filter.
func Reject(job dto.Job, cfg dto.SearchConfig) (reason string, rejected bool) {
	if job.CompanySlug != "" {
		for _, excluded := range cfg.ExcludedCompanies {
			if slug.Make(excluded) == job.CompanySlug {
				return fmt.Sprintf("company: %s", job.CompanySlug), true
			}
		}
	}

	contentTokens := tokenize(job.Title + " " + job.Description)
	for _, tech := range cfg.Preferences.BlockedTech {
		if hasPhrase(contentTokens, tech) {
			return fmt.Sprintf("blocked tech: %s", strings.ToLower(tech)), true
		}
	}

	return "", false
}

// RejectLocation reports whether job trips cfg's excluded-locations filter.
func RejectLocation(job dto.Job, cfg dto.SearchConfig) (reason string, rejected bool) {
	locationTokens := tokenize(job.Location)
	for _, loc := range cfg.ExcludedLocations {
		if hasPhrase(locationTokens, loc) {
			return fmt.Sprintf("location: %s", strings.ToLower(loc)), true
		}
	}
	return "", false
}

func tokenize(s string) []string {
	var tokens []string
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '+' || r == '#':
			b.WriteRune(r)
		default:
			if b.Len() > 0 {
				tokens = append(tokens, b.String())
				b.Reset()
			}
		}
	}
	if b.Len() > 0 {
		tokens = append(tokens, b.String())
	}
	return tokens
}

func hasPhrase(tokens []string, phrase string) bool {
	phraseTokens := tokenize(phrase)
	if len(phraseTokens) == 0 || len(phraseTokens) > len(tokens) {
		return false
	}
	for i := 0; i+len(phraseTokens) <= len(tokens); i++ {
		match := true
		for j, pt := range phraseTokens {
			if tokens[i+j] != pt {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

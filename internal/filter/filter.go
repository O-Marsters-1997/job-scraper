package filter

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/slug"
)

// Reject reports whether job trips any of cfg's exclusion filters or matches
// none of a non-empty include list. A blank or bare "Remote" location passes
// the location include list.
func Reject(job dto.Job, cfg dto.SearchConfig) (reason string, rejected bool) {
	if job.CompanySlug != "" {
		jobCompany := slug.Company(job.CompanySlug)
		for _, excluded := range cfg.ExcludedCompanies {
			if slug.Company(excluded) == jobCompany {
				return fmt.Sprintf("company: %s", job.CompanySlug), true
			}
		}
	}

	titleTokens := tokenize(job.Title)
	for _, kw := range cfg.ExcludedTitleKeywords {
		if hasPhrase(titleTokens, kw) {
			return fmt.Sprintf("title keyword: %s", strings.ToLower(kw)), true
		}
	}

	locationTokens := tokenize(job.Location)
	for _, loc := range cfg.ExcludedLocations {
		if hasPhrase(locationTokens, loc) {
			return fmt.Sprintf("location: %s", strings.ToLower(loc)), true
		}
	}

	if len(cfg.RequiredTitleKeywords) > 0 && !hasAnyPhrase(titleTokens, cfg.RequiredTitleKeywords) {
		return "title: no required keyword", true
	}

	if len(cfg.RequiredLocations) > 0 && !isUnspecifiedLocation(locationTokens) &&
		!hasAnyPhrase(locationTokens, cfg.RequiredLocations) {
		return "location: no required location", true
	}

	return "", false
}

func isUnspecifiedLocation(tokens []string) bool {
	return len(tokens) == 0 || (len(tokens) == 1 && tokens[0] == "remote")
}

func hasAnyPhrase(tokens []string, phrases []string) bool {
	for _, p := range phrases {
		if hasPhrase(tokens, p) {
			return true
		}
	}
	return false
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

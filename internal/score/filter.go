package score

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

// SeniorityLevels are the canonical values accepted in SearchConfig.ExcludedSeniority.
var SeniorityLevels = []string{
	"intern", "junior", "mid", "senior", "staff", "principal", "lead", "manager", "director",
}

var senioritySignals = map[string][]string{
	"intern":    {"intern", "internship", "trainee", "apprentice"},
	"junior":    {"junior", "jr", "graduate", "entry level"},
	"mid":       {"mid level", "midlevel"},
	"senior":    {"senior", "sr"},
	"staff":     {"staff"},
	"principal": {"principal"},
	"lead":      {"lead"},
	"manager":   {"manager"},
	"director":  {"director", "head of", "vp", "vice president", "chief", "cto"},
}

// Reject reports whether job trips any of cfg's exclusion filters.
func Reject(job dto.Job, cfg dto.SearchConfig) (reason string, rejected bool) {
	if job.CompanySlug != "" {
		for _, excluded := range cfg.ExcludedCompanies {
			if sources.Slugify(excluded) == job.CompanySlug {
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

	for _, level := range cfg.ExcludedSeniority {
		for _, signal := range senioritySignals[level] {
			if hasPhrase(titleTokens, signal) {
				return fmt.Sprintf("seniority: %s", level), true
			}
		}
	}

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

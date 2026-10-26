package lineedit

import (
	_ "embed"
	"fmt"
	"maps"
	"slices"
	"unicode/utf8"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
)

const (
	editModel    = "anthropic/claude-haiku-4.5"
	actionTailor = dto.SuggestActionTailor
)

//go:embed suggest.md
var suggestRules string

var systemPrompt = cvedit.Voice + "\n" + suggestRules

type action struct {
	render func(Request) string
	reject func(req Request, text string) string
}

func noRule(Request, string) string { return "" }

var actions = map[string]action{
	"fit": {
		render: func(r Request) string {
			return fmt.Sprintf("Shorten the line to at most %d characters, keeping its meaning and the facts it draws on.\n", r.MaxChars)
		},
		reject: func(r Request, text string) string {
			if utf8.RuneCountInString(text) > r.MaxChars {
				return fmt.Sprintf("the line is longer than %d characters", r.MaxChars)
			}
			return ""
		},
	},
	"tighten": {
		render: func(Request) string {
			return "Tighten the line: cut filler words and keep every fact. It must not get longer.\n"
		},
		reject: func(r Request, text string) string {
			if utf8.RuneCountInString(text) > utf8.RuneCountInString(r.Text) {
				return "the tightened line is longer than the original"
			}
			return ""
		},
	},
	"verb": {
		render: func(Request) string {
			return "Start the line with a stronger, more specific verb. Change nothing else.\n"
		},
		reject: noRule,
	},
	actionTailor: {
		render: func(r Request) string {
			return fmt.Sprintf("Swap only the nouns and terms in the line that differ from the wording of this quote from the job description, keeping the same sentence shape, verb and voice. Never add a claim the Achievements lack, and never add words.\n\nQuote:\n%s\n", r.Quote)
		},
		reject: func(r Request, text string) string {
			limit := utf8.RuneCountInString(r.Text)
			if r.MaxChars > 0 {
				limit = min(limit, r.MaxChars)
			}
			return tailorViolation(r.Text, text, limit, dto.TailorMaxChangedShare)
		},
	},
	"ask": {
		render: func(r Request) string {
			return fmt.Sprintf("Rewrite the line as the person asks, within the rules: %s\n", r.Prompt)
		},
		reject: noRule,
	},
}

var actionNames = slices.Sorted(maps.Keys(actions))

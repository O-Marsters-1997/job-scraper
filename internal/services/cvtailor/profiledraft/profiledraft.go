package profiledraft

import (
	"context"
	_ "embed"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/lineedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/linefit"
)

const (
	redraftModel       = "anthropic/claude-sonnet-4.5"
	redraftMaxTokens   = 1000
	maxJobRunes        = 6000
	profileSlotID      = "profile"
	msgNotOneParagraph = "the profile must be exactly one paragraph"
	fallbackWidthPt    = 468
	maxNewWords        = 12
)

//go:embed profile.md
var profileRules string

var systemPrompt = cvedit.Voice + "\n" + profileRules

type Input struct {
	Profile        string
	Bank           []string
	JobTitle       string
	Company        string
	JobDescription string
	Steer          string
	Lines          func(string) int
}

type Attempt struct {
	Text      string
	Steer     string
	Rejection string
	Overlong  bool
	Findings  []checks.Finding
	Cost      float64
	Model     string
	PromptKey string
}

type Result struct {
	Text     string
	Overlong bool
	Attempts []Attempt
}

type Writer struct{ model lineedit.Model }

func New(model lineedit.Model) *Writer { return &Writer{model: model} }

func (w *Writer) Redraft(ctx context.Context, apiKey string, in Input) (Result, error) {
	base := userPrompt(in)
	user := base
	var res Result
	lines := defaultLines
	if in.Lines != nil {
		lines = in.Lines
	}
	budget := lines(in.Profile)
	for range 2 {
		p := lineedit.Prompt{Model: redraftModel, System: systemPrompt, User: user, MaxTokens: redraftMaxTokens}
		c, err := w.model.Complete(ctx, apiKey, p, func(string) {})
		if err != nil {
			return res, fmt.Errorf("redraft profile: %w", err)
		}
		text := strings.TrimSpace(c.Text)
		found := guard(in, text)
		a := Attempt{
			Text: text, Steer: in.Steer, Findings: found, Cost: c.Cost,
			Model: p.Model, PromptKey: p.Key(), Rejection: rejection(in.Profile, text, found),
		}
		if n := lines(text); a.Rejection == "" && n > budget {
			a.Overlong = true
			a.Rejection = fmt.Sprintf("%d lines is over the %d-line budget", n, budget)
		}
		res.Attempts = append(res.Attempts, a)
		if a.Rejection == "" {
			res.Text = text
			return res, nil
		}
		if a.Overlong {
			user = fmt.Sprintf("%s\nYour previous rewrite:\n%s\n\nIt is too long. Shorten that exact text to at most %d words. Keep its order and facts, add nothing and do not start again.\n", base, a.Text, len(strings.Fields(in.Profile)))
			continue
		}
		user = fmt.Sprintf("%s\nYour previous rewrite:\n%s\n\nIt was rejected: %s\nWrite a new version that fixes this.\n", base, a.Text, a.Rejection)
	}
	if last := res.Attempts[len(res.Attempts)-1]; last.Overlong {
		res.Text, res.Overlong = last.Text, true
	}
	return res, nil
}

func defaultLines(text string) int {
	return linefit.Lines(text, linefit.Style{Font: "Arial", Size: 11}, fallbackWidthPt)
}

func userPrompt(in Input) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Job: %s at %s\n\nJob description:\n%s\n\nCurrent profile:\n%s\n",
		in.JobTitle, in.Company, JobText(in.JobDescription), in.Profile)
	fmt.Fprintf(&b, "\nBudget: at most %d words, the length of the current profile. Introduce at most %d words that are not already in it.\n", len(strings.Fields(in.Profile)), maxNewWords)
	if steer := strings.TrimSpace(in.Steer); steer != "" {
		fmt.Fprintf(&b, "The person asks: %s\nThe steer reorders and re-emphasises content within the budget; it does not add length.\n", steer)
	}
	return b.String()
}

func guard(in Input, text string) []checks.Finding {
	d := checks.Draft{
		Profile: &checks.Slot{ID: profileSlotID, Text: text, BaseText: in.Profile},
		Bank:    in.Bank,
	}
	return checks.Blocking(slices.Concat(checks.Grounding(d), checks.BannedWords(d)))
}

func rejection(base, text string, found []checks.Finding) string {
	var msgs []string
	if text == "" || strings.Contains(text, "\n") {
		msgs = append(msgs, msgNotOneParagraph)
	}
	if n := newWords(base, text); n > maxNewWords {
		msgs = append(msgs, fmt.Sprintf("%d words are new against the current profile, at most %d allowed; keep its wording and swap only a few phrases", n, maxNewWords))
	}
	for _, f := range found {
		msgs = append(msgs, f.Message)
	}
	return strings.Join(msgs, "; ")
}

func newWords(base, text string) int {
	have := map[string]bool{}
	for _, w := range words(base) {
		have[w] = true
	}
	n := 0
	for _, w := range words(text) {
		if !have[w] {
			n++
		}
	}
	return n
}

func words(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func JobText(html string) string {
	text := html
	if doc, err := goquery.NewDocumentFromReader(strings.NewReader(html)); err == nil {
		text = strings.TrimSpace(doc.Text())
	}
	if r := []rune(text); len(r) > maxJobRunes {
		return string(r[:maxJobRunes])
	}
	return text
}

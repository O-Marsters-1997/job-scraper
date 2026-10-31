package profiledraft

import (
	"context"
	_ "embed"
	"fmt"
	"slices"
	"strings"

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
)

//go:embed profile.md
var profileRules string

var systemPrompt = cvedit.Voice + "\n" + profileRules

type Input struct {
	Profile        string
	BankSkills     []string
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
			Model: p.Model, PromptKey: p.Key(), Rejection: rejection(text, found),
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
	fmt.Fprintf(&b, "Job: %s at %s\n\nJob description:\n%s\n\nCurrent profile:\n%s\n\nSkills the person has:\n",
		in.JobTitle, in.Company, JobText(in.JobDescription), in.Profile)
	for _, s := range in.BankSkills {
		fmt.Fprintf(&b, "- %s\n", s)
	}
	fmt.Fprintf(&b, "\nBudget: at most %d words, the length of the current profile.\n", len(strings.Fields(in.Profile)))
	if steer := strings.TrimSpace(in.Steer); steer != "" {
		fmt.Fprintf(&b, "The person asks: %s\nThe steer reorders and re-emphasises content within the budget; it does not add length.\n", steer)
	}
	return b.String()
}

func guard(in Input, text string) []checks.Finding {
	d := checks.Draft{
		Profile:    &checks.Slot{ID: profileSlotID, Text: text, BaseText: in.Profile},
		Bank:       in.Bank,
		BankSkills: in.BankSkills,
	}
	return checks.Blocking(slices.Concat(checks.Grounding(d), checks.BannedWords(d)))
}

func rejection(text string, found []checks.Finding) string {
	var msgs []string
	if text == "" || strings.Contains(text, "\n") {
		msgs = append(msgs, msgNotOneParagraph)
	}
	for _, f := range found {
		msgs = append(msgs, f.Message)
	}
	return strings.Join(msgs, "; ")
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

package profiledraft

import (
	"context"
	_ "embed"
	"fmt"
	"slices"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/ollymarsters/job-scraper/internal/promptparts"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
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

//go:embed parts/redraft-profile.md
var redraftProfilePart string

//go:embed parts/voice.md
var voicePart string

var Generation = promptparts.Generation{
	Name: promptparts.ProfileRedraft,
	Main: "redraft-profile.md",
	Parts: []promptparts.Part{
		{Path: "redraft-profile.md", Default: redraftProfilePart},
		{Path: "voice.md", Default: voicePart},
	},
}

type Input struct {
	Profile        string
	Bank           []string
	BankSkills     []string
	JobTitle       string
	Company        string
	JobDescription string
	Steer          string
	Parts          map[string]string
	Lines          func(string) int
}

func Documents(in Input) []promptparts.Document {
	return []promptparts.Document{
		{Path: "job.md", Text: fmt.Sprintf("# %s at %s\n\n%s", in.JobTitle, in.Company, JobText(in.JobDescription))},
		{Path: "profile.md", Text: fmt.Sprintf("---\nbudget_words: %d\n---\n\n%s", len(strings.Fields(in.Profile)), in.Profile)},
	}
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
	base := promptparts.Message(Generation, in.Parts, Documents(in), in.Steer)
	user := base
	var res Result
	lines := defaultLines
	if in.Lines != nil {
		lines = in.Lines
	}
	budget := lines(in.Profile)
	for range 2 {
		p := lineedit.Prompt{Model: redraftModel, System: promptparts.System, User: user, MaxTokens: redraftMaxTokens}
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

type TryResult struct {
	Text      string
	Findings  []checks.Finding
	Cost      float64
	Model     string
	PromptKey string
}

func tryPrompt(in Input) lineedit.Prompt {
	return lineedit.Prompt{
		Model: redraftModel, System: promptparts.System, MaxTokens: redraftMaxTokens,
		User: promptparts.Message(Generation, in.Parts, Documents(in), in.Steer),
	}
}

type UseResult struct {
	Overlong bool
	Attempt  Attempt
}

func Use(in Input, text string) UseResult {
	text = strings.TrimSpace(text)
	p := tryPrompt(in)
	found := guard(in, text)
	lines := defaultLines
	if in.Lines != nil {
		lines = in.Lines
	}
	return UseResult{
		Overlong: lines(text) > lines(in.Profile),
		Attempt: Attempt{
			Text: text, Steer: in.Steer, Findings: found, Model: p.Model,
			PromptKey: p.Key(), Rejection: rejection(text, found),
		},
	}
}

func (w *Writer) Try(ctx context.Context, apiKey string, in Input, onDelta func(string)) (TryResult, error) {
	p := tryPrompt(in)
	res := TryResult{Model: p.Model, PromptKey: p.Key()}
	c, err := w.model.Complete(ctx, apiKey, p, onDelta)
	res.Cost = c.Cost
	if err != nil {
		return res, fmt.Errorf("try profile: %w", err)
	}
	res.Text = strings.TrimSpace(c.Text)
	res.Findings = guard(in, res.Text)
	return res, nil
}

func defaultLines(text string) int {
	return linefit.Lines(text, linefit.Style{Font: "Arial", Size: 11}, fallbackWidthPt)
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

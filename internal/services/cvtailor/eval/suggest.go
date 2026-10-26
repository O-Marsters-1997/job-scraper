package eval

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/lineedit"
)

// SuggestFixture is one inline-edit request: a line, the Achievements it may
// draw on and the action asked of the model.
type SuggestFixture struct {
	Name         string
	Scenario     string
	Text         string
	Achievements []string
	Scope        []string
	Bank         []string
	Profile      bool
	Action       string
	Prompt       string
	Quote        string
	MaxChars     int
}

func SuggestFixtures() ([]SuggestFixture, error) {
	return loadFixtures("fixtures/suggest", func(f *SuggestFixture) *string { return &f.Name })
}

type SuggestOutcome struct {
	Fixture   string
	Action    string
	Output    string
	Findings  []checks.Finding
	Rejection string
	Cost      float64
	Model     string
	PromptKey string
}

// Passed reports whether the suggestion is grounded, free of banned words and
// within the length its action allows.
func (o SuggestOutcome) Passed() bool {
	return len(checks.Blocking(o.Findings)) == 0 && o.Rejection == ""
}

func RunSuggest(ctx context.Context, e *lineedit.Editor, apiKey string, f SuggestFixture) (SuggestOutcome, error) {
	req := lineedit.Request{Action: f.Action, Prompt: f.Prompt, Quote: f.Quote, Text: f.Text, MaxChars: f.MaxChars, SlotID: "s0"}
	ev := lineedit.Evidence{
		Cited:   f.Achievements,
		Scope:   orElse(f.Scope, f.Achievements),
		Bank:    orElse(f.Bank, f.Achievements),
		Profile: f.Profile,
	}
	res, err := e.Rewrite(ctx, apiKey, req, ev, func(string) {})
	out := SuggestOutcome{
		Fixture: f.Name, Action: f.Action, Output: res.Text, Findings: res.Findings,
		Rejection: res.Rejection, Cost: res.Cost, Model: res.Model, PromptKey: res.PromptKey,
	}
	if err != nil {
		return out, fmt.Errorf("suggest fixture %s: %w", f.Name, err)
	}
	return out, nil
}

func orElse(v, fallback []string) []string {
	if len(v) > 0 {
		return v
	}
	return fallback
}

func SuggestReport(outcomes []SuggestOutcome) string {
	var b strings.Builder
	model := ""
	if len(outcomes) > 0 {
		model = outcomes[0].Model
	}
	fmt.Fprintf(&b, "suggest model=%s runs=%d\n\n", model, len(outcomes))
	passed, cost := 0, 0.0
	perFixture := map[string][2]int{}
	keys := map[string]string{}
	for _, o := range outcomes {
		cost += o.Cost
		keys[o.Fixture] = o.PromptKey
		c := perFixture[o.Fixture]
		c[1]++
		if o.Passed() {
			passed++
			c[0]++
		}
		perFixture[o.Fixture] = c
	}
	fmt.Fprintf(&b, "pass %s\n", rate(passed, len(outcomes)))
	for _, name := range slices.Sorted(maps.Keys(perFixture)) {
		c := perFixture[name]
		fmt.Fprintf(&b, "%s: %s prompt_key=%s\n", name, rate(c[0], c[1]), keys[name])
	}
	for _, o := range outcomes {
		if o.Passed() {
			continue
		}
		fmt.Fprintf(&b, "\nFAIL %s: %q\n", o.Fixture, o.Output)
		for _, f := range checks.Blocking(o.Findings) {
			fmt.Fprintf(&b, "  [%s] %s\n", f.Check, f.Message)
		}
		if o.Rejection != "" {
			fmt.Fprintf(&b, "  rejected: %s\n", o.Rejection)
		}
	}
	fmt.Fprintf(&b, "\ntotal cost=$%.4f\n", cost)
	return b.String()
}

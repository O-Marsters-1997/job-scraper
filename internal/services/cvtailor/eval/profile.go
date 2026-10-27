package eval

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/profiledraft"
)

type ProfileFixture struct {
	Name           string
	Scenario       string
	Profile        string
	BankSkills     []string
	Bank           []string
	JobTitle       string
	Company        string
	JobDescription string
	Steer          string
}

func ProfileFixtures() ([]ProfileFixture, error) {
	return loadFixtures("fixtures/profile", func(f *ProfileFixture) *string { return &f.Name })
}

type ProfileOutcome struct {
	Fixture   string
	Output    string
	Attempts  int
	Findings  []checks.Finding
	Rejection string
	Cost      float64
	Model     string
	PromptKey string
}

func (o ProfileOutcome) Passed() bool {
	return o.Output != "" && o.Rejection == ""
}

func RunProfile(ctx context.Context, w *profiledraft.Writer, apiKey string, f ProfileFixture) (ProfileOutcome, error) {
	res, err := w.Redraft(ctx, apiKey, profiledraft.Input{
		Profile: f.Profile, BankSkills: f.BankSkills, Bank: f.Bank, JobTitle: f.JobTitle,
		Company: f.Company, JobDescription: f.JobDescription, Steer: f.Steer,
	})
	out := ProfileOutcome{Fixture: f.Name, Output: res.Text, Attempts: len(res.Attempts)}
	for _, a := range res.Attempts {
		out.Cost += a.Cost
	}
	if n := len(res.Attempts); n > 0 {
		last := res.Attempts[n-1]
		out.Findings, out.Rejection = last.Findings, last.Rejection
		out.Model, out.PromptKey = res.Attempts[0].Model, res.Attempts[0].PromptKey
		if out.Output == "" {
			out.Output = last.Text
		}
	}
	if err != nil {
		return out, fmt.Errorf("profile fixture %s: %w", f.Name, err)
	}
	return out, nil
}

func ProfileReport(outcomes []ProfileOutcome) string {
	var b strings.Builder
	model := ""
	if len(outcomes) > 0 {
		model = outcomes[0].Model
	}
	fmt.Fprintf(&b, "profile model=%s runs=%d\n\n", model, len(outcomes))
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
		for _, f := range o.Findings {
			fmt.Fprintf(&b, "  [%s] %s\n", f.Check, f.Message)
		}
		if o.Rejection != "" {
			fmt.Fprintf(&b, "  rejected: %s\n", o.Rejection)
		}
	}
	fmt.Fprintf(&b, "\ntotal cost=$%.4f\n", cost)
	return b.String()
}

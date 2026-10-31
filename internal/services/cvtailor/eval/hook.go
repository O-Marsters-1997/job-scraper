package eval

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/lineedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/outreachhook"
)

type HookFixture struct {
	Name           string
	Scenario       string
	JobTitle       string
	JobDescription string
	Company        string
	ContactRole    string
	Achievement    string
}

func HookFixtures() ([]HookFixture, error) {
	return loadFixtures("fixtures/hook", func(f *HookFixture) *string { return &f.Name })
}

type HookOutcome struct {
	Fixture   string
	Output    string
	Findings  []dto.LetterFinding
	Cost      float64
	PromptKey string
}

func (o HookOutcome) Passed() bool {
	return o.Output != "" && len(o.Findings) == 0
}

type keyRecorder struct {
	lineedit.Model
	key string
}

func (k *keyRecorder) Complete(ctx context.Context, apiKey string, p lineedit.Prompt, onDelta func(string)) (lineedit.Completion, error) {
	k.key = p.Key()
	return k.Model.Complete(ctx, apiKey, p, onDelta)
}

func RunHook(ctx context.Context, model lineedit.Model, apiKey string, f HookFixture) (HookOutcome, error) {
	rec := &keyRecorder{Model: model}
	res, err := outreachhook.New(rec).Write(ctx, apiKey, outreachhook.Input{
		JobTitle: f.JobTitle, JobDescription: f.JobDescription, Company: f.Company,
		ContactRole: f.ContactRole, Achievement: f.Achievement,
	})
	out := HookOutcome{Fixture: f.Name, Output: res.Text, Cost: res.Cost, PromptKey: rec.key}
	if err != nil {
		return out, fmt.Errorf("hook fixture %s: %w", f.Name, err)
	}
	out.Findings = checks.Hook(res.Text, f.Company)
	return out, nil
}

func HookReport(outcomes []HookOutcome) string {
	var b strings.Builder
	fmt.Fprintf(&b, "hook model=%s runs=%d\n\n", outreachhook.Model, len(outcomes))
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
			fmt.Fprintf(&b, "  [%s] %s\n", f.Kind, f.Text)
		}
	}
	fmt.Fprintf(&b, "\ntotal cost=$%.4f\n", cost)
	return b.String()
}

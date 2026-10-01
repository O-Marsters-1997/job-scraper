package eval_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/eval"
)

func suggestFixture(t *testing.T, name string) eval.SuggestFixture {
	t.Helper()
	all, err := eval.SuggestFixtures()
	if err != nil {
		t.Fatalf("SuggestFixtures: %v", err)
	}
	for _, f := range all {
		if f.Name == name {
			return f
		}
	}
	t.Fatalf("no suggest fixture %q", name)
	return eval.SuggestFixture{}
}

func TestSuggestFixturesCoverEveryActionAndEmbellishment(t *testing.T) {
	all, err := eval.SuggestFixtures()
	if err != nil {
		t.Fatalf("SuggestFixtures: %v", err)
	}
	var actions []string
	for _, f := range all {
		if f.Scenario == "" || f.Text == "" || len(f.Achievements) == 0 {
			t.Errorf("suggest fixture %s is incomplete", f.Name)
		}
		actions = append(actions, f.Action)
	}
	for _, want := range []string{"fit", "tighten", "ground", "verb", "ask"} {
		if !slices.Contains(actions, want) {
			t.Errorf("no suggest fixture for action %q", want)
		}
	}
	if !slices.ContainsFunc(all, func(f eval.SuggestFixture) bool { return f.Name == "ask-embellish-metric" }) {
		t.Error("no ask fixture that tempts a made-up metric")
	}
}

func TestRunSuggestScoresGroundingBannedWordsAndLength(t *testing.T) {
	cases := []struct {
		name    string
		fixture string
		output  string
		want    bool
	}{
		{"grounded output passes", "ask-embellish-metric", "Moved the billing service to Kubernetes", true},
		{"a made-up number fails", "ask-embellish-metric", "Moved the billing service to Kubernetes, 3x faster", false},
		{"a banned word fails", "verb-weak", "Spearheaded the weekly release train for six squads", false},
		{"a fit over the limit fails", "fit-latency", strings.Repeat("Cut p99 latency ", 10), false},
		{"a tighten that grows fails", "tighten-onboarding", "Mentored four junior engineers through their very first on-call rotation, and also wrote the runbook that they used, every time", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := suggestFixture(t, tc.fixture)

			out, err := eval.RunSuggest(t.Context(), cvtailortest.Suggesting(tc.output), "key", f)

			if err != nil {
				t.Fatalf("RunSuggest: %v", err)
			}
			if out.Passed() != tc.want {
				t.Errorf("RunSuggest(%s, %q).Passed() = %t, want %t (findings %+v, over limit %t)", tc.fixture, tc.output, out.Passed(), tc.want, out.Findings, out.OverLimit)
			}
		})
	}
}

func TestSuggestReportShowsPassRatePerFixture(t *testing.T) {
	f := suggestFixture(t, "verb-weak")
	good, _ := eval.RunSuggest(t.Context(), cvtailortest.Suggesting("Ran the weekly release train for six squads"), "key", f)
	bad, _ := eval.RunSuggest(t.Context(), cvtailortest.Suggesting("Spearheaded the weekly release train for six squads"), "key", f)

	report := eval.SuggestReport("v1", "m", []eval.SuggestOutcome{good, bad})

	for _, want := range []string{"prompt_version=v1", "verb-weak: 1/2 (50%)", "pass 1/2 (50%)"} {
		if !strings.Contains(report, want) {
			t.Errorf("report missing %q:\n%s", want, report)
		}
	}
}

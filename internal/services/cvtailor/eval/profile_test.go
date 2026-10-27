package eval_test

import (
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/eval"
)

func TestProfileFixturesCoverTemptations(t *testing.T) {
	t.Parallel()
	all, err := eval.ProfileFixtures()
	if err != nil {
		t.Fatalf("ProfileFixtures: %v", err)
	}
	have := map[string]bool{}
	for _, f := range all {
		if f.Scenario == "" || f.Profile == "" || f.JobDescription == "" {
			t.Errorf("profile fixture %s is incomplete", f.Name)
		}
		have[f.Name] = true
	}
	for _, want := range []string{"invented-metric", "invented-technology", "steer-unsupported-fact", "over-length-base"} {
		if !have[want] {
			t.Errorf("no profile fixture %q", want)
		}
	}
}

func TestProfileReport(t *testing.T) {
	t.Parallel()
	outcomes := []eval.ProfileOutcome{
		{Fixture: "a", Output: "ok", Attempts: 1, Cost: 0.01, Model: "m", PromptKey: "k1"},
		{
			Fixture: "b", Output: "bad", Attempts: 2, Cost: 0.02, Model: "m", PromptKey: "k2",
			Rejection: `number "40%" does not appear`,
			Findings:  []checks.Finding{{Check: "grounding", Severity: checks.Block, Message: `number "40%" does not appear`}},
		},
	}
	got := eval.ProfileReport(outcomes)
	for _, want := range []string{"pass 1/2 (50%)", "a: 1/1 (100%) prompt_key=k1", "FAIL b", "[grounding]", "total cost=$0.0300"} {
		if !strings.Contains(got, want) {
			t.Errorf("ProfileReport lacks %q:\n%s", want, got)
		}
	}
}

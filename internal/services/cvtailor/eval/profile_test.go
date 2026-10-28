package eval_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/eval"
)

func TestProfileFixtures(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "missing")

	t.Run("synthetic fixtures cover temptations", func(t *testing.T) {
		t.Parallel()
		all, err := eval.ProfileFixtures(missing)
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
	})

	t.Run("local fixtures are added to the synthetic ones", func(t *testing.T) {
		t.Parallel()
		synthetic, err := eval.ProfileFixtures(missing)
		if err != nil {
			t.Fatalf("ProfileFixtures: %v", err)
		}
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "mine.json"), []byte(`{"Profile":"p","Final":"f"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := eval.ProfileFixtures(dir)
		if err != nil {
			t.Fatalf("ProfileFixtures: %v", err)
		}
		if len(got) != len(synthetic)+1 {
			t.Fatalf("len = %d, want %d", len(got), len(synthetic)+1)
		}
		i := slices.IndexFunc(got, func(f eval.ProfileFixture) bool { return f.Name == "mine" })
		if i < 0 || got[i].Final != "f" {
			t.Errorf("local fixture mine with final f not found in %+v", got)
		}
	})
}

func TestProfileReport(t *testing.T) {
	t.Parallel()

	t.Run("pass rate and failures", func(t *testing.T) {
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
	})

	t.Run("edit burden mean and output beside final", func(t *testing.T) {
		t.Parallel()
		burden := func(v float64) *float64 { return &v }
		got := eval.ProfileReport([]eval.ProfileOutcome{
			{Fixture: "a", Output: "mine out", Final: "kept final", EditBurden: burden(0.5), Model: "m"},
			{Fixture: "b", Output: "o2", Final: "f2", EditBurden: burden(0.25), Model: "m"},
			{Fixture: "synthetic", Output: "x", Model: "m"},
		})
		for _, want := range []string{"pass 3/3", "mean edit burden 0.38 over 2", "BURDEN a: 0.50", `output: "mine out"`, `final:  "kept final"`} {
			if !strings.Contains(got, want) {
				t.Errorf("ProfileReport lacks %q:\n%s", want, got)
			}
		}
	})

	t.Run("no mean without a final", func(t *testing.T) {
		t.Parallel()
		got := eval.ProfileReport([]eval.ProfileOutcome{{Fixture: "a", Output: "ok", Model: "m"}})
		if strings.Contains(got, "edit burden") {
			t.Errorf("ProfileReport mentions edit burden:\n%s", got)
		}
	})
}

package eval_test

import (
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/eval"
)

func bullet(text string, ids ...string) cvedit.EditSet {
	return cvedit.EditSet{Positions: []cvedit.PositionEdit{{
		PositionID: "p1",
		Bullets:    []cvedit.Bullet{{AchievementIDs: ids, Text: text}},
	}}}
}

func fixture(t *testing.T, name string) eval.Fixture {
	t.Helper()
	all, err := eval.Fixtures()
	if err != nil {
		t.Fatalf("Fixtures: %v", err)
	}
	for _, f := range all {
		if f.Name == name {
			return f
		}
	}
	t.Fatalf("no fixture %q", name)
	return eval.Fixture{}
}

func TestFixturesCoverPRDScenarios(t *testing.T) {
	all, err := eval.Fixtures()
	if err != nil {
		t.Fatalf("Fixtures: %v", err)
	}
	if len(all) != 7 {
		t.Fatalf("Fixtures() = %d fixtures, want the four PRD scenarios and three term fixtures", len(all))
	}
	for _, f := range all {
		if f.Scenario == "" || f.JobDescription == "" || len(f.Positions) == 0 {
			t.Errorf("fixture %s is incomplete", f.Name)
		}
	}
}

func TestRunRetriesOnBlockFindingAndReports(t *testing.T) {
	f := fixture(t, "embellish-temptation")
	ed := cvtailortest.Editing(
		cvedit.Result{Edits: bullet("Cut API latency by 40%", "a1"), Cost: 0.01},
		cvedit.Result{Edits: bullet("Cut API latency", "a1"), Cost: 0.01},
	)

	out, err := eval.Run(t.Context(), ed, "key", f)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if out.Retries() != 1 {
		t.Errorf("Retries() = %d, want 1", out.Retries())
	}
	if len(ed.Inputs[1].PriorFindings) == 0 || ed.Inputs[1].PriorEdits == nil {
		t.Errorf("retry input = %+v, want prior edits and findings", ed.Inputs[1])
	}

	report := eval.Report("v1", "m", []eval.Outcome{out})
	for _, want := range []string{"prompt_version=v1", "total retries=1"} {
		if !strings.Contains(report, want) {
			t.Errorf("report missing %q:\n%s", want, report)
		}
	}
	var grounding []string
	for line := range strings.SplitSeq(report, "\n") {
		if strings.HasPrefix(line, "grounding") {
			grounding = strings.Fields(line)
		}
	}
	if got := strings.Join(grounding, " "); got != "grounding 0/1 (0%) 1/1 (100%)" {
		t.Errorf("grounding row = %q, want it to fail first and pass after the retry", got)
	}
}

func TestRunStopsAfterTwoRetries(t *testing.T) {
	f := fixture(t, "embellish-temptation")
	ed := cvtailortest.Editing(cvedit.Result{Edits: bullet("Cut API latency by 40%", "a1")})

	out, err := eval.Run(t.Context(), ed, "key", f)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Retries() != 2 {
		t.Errorf("Retries() = %d, want 2", out.Retries())
	}
}

func TestRunReportsTermCoverage(t *testing.T) {
	f := fixture(t, "term-synonym-trap")
	ed := cvtailortest.Editing(cvedit.Result{Edits: bullet("Ran nightly jobs on Airflow", "a1")})

	out, err := eval.Run(t.Context(), ed, "key", f)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if out.TermsSupported != 2 || out.TermsUsed != 1 {
		t.Errorf("terms supported/used = %d/%d, want 2/1 (dbt and Spark are not in the bank)", out.TermsSupported, out.TermsUsed)
	}
	if report := eval.Report("v1", "m", []eval.Outcome{out}); !strings.Contains(report, "term_coverage 1/2 (50%)") {
		t.Errorf("report missing term_coverage:\n%s", report)
	}
}

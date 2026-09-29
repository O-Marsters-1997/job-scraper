package eval_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/tailoring/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/tailoring/eval"
)

type scriptedEditor struct {
	results []cvedit.Result
	inputs  []cvedit.Input
}

func (e *scriptedEditor) Edit(_ context.Context, _ string, in cvedit.Input) (cvedit.Result, error) {
	e.inputs = append(e.inputs, in)
	r := e.results[min(len(e.inputs), len(e.results))-1]
	return r, nil
}

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

func TestFixturesCoverPRDScenariosAndHaveSlots(t *testing.T) {
	all, err := eval.Fixtures()
	if err != nil {
		t.Fatalf("Fixtures: %v", err)
	}
	if len(all) != 4 {
		t.Fatalf("Fixtures() = %d fixtures, want the four PRD scenarios", len(all))
	}
	for _, f := range all {
		if f.Scenario == "" || f.JobDescription == "" || len(f.Positions) == 0 {
			t.Errorf("fixture %s is incomplete", f.Name)
		}
	}
}

func TestRun_RetriesOnBlockFindingAndReports(t *testing.T) {
	f := fixture(t, "embellish-temptation")
	ed := &scriptedEditor{results: []cvedit.Result{
		{Edits: bullet("Cut API latency by 40%", "a1"), Cost: 0.01},
		{Edits: bullet("Cut API latency", "a1"), Cost: 0.01},
	}}

	out, err := eval.Run(context.Background(), ed, "key", f)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if out.Retries() != 1 {
		t.Errorf("Retries() = %d, want 1", out.Retries())
	}
	if len(ed.inputs[1].PriorFindings) == 0 || ed.inputs[1].PriorEdits == nil {
		t.Errorf("retry input = %+v, want prior edits and findings", ed.inputs[1])
	}

	var buf bytes.Buffer
	eval.Report(&buf, "v1", "m", []eval.Outcome{out})
	report := buf.String()
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

func TestRun_StopsAfterTwoRetries(t *testing.T) {
	f := fixture(t, "embellish-temptation")
	ed := &scriptedEditor{results: []cvedit.Result{{Edits: bullet("Cut API latency by 40%", "a1")}}}

	out, err := eval.Run(context.Background(), ed, "key", f)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Retries() != 2 {
		t.Errorf("Retries() = %d, want 2", out.Retries())
	}
}

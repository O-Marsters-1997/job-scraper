package scoring_test

import (
	"flag"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
)

var update = flag.Bool("update", false, "rewrite golden files")

func TestExportFeedback(t *testing.T) {
	const userID = "user-1"
	st := newFakeStore()
	seedScoredJob(st, userID, "tech:go", "tech:cobol", "tech:kubernetes")
	m := scoring.Build(newDeps(t, st))
	if _, err := scoring.NewService(newDeps(t, st)).AppendJobFeedback(t.Context(), userID, dto.JobFeedbackInput{JobID: testJob.ID, Direction: "lower", Reason: "Go is a given here."}); err != nil {
		t.Fatalf("AppendJobFeedback() err = %v", err)
	}

	if _, err := scoring.NewService(newDeps(t, st)).AppendCollectionFeedback(t.Context(), userID, dto.CollectionFeedbackInput{
		JobIDs: []string{"missing", testJob.ID}, Filters: map[string]string{"q": "go", "scored": "true"}, Reason: "The ranking is off.",
	}); err != nil {
		t.Fatalf("AppendCollectionFeedback() err = %v", err)
	}

	for _, reason := range []string{"Scores run hot for backend roles.", "Two lines\nof reasoning."} {
		if _, err := scoring.NewService(newDeps(t, st)).AppendOverallFeedback(t.Context(), userID, dto.OverallFeedbackInput{Reason: reason}); err != nil {
			t.Fatalf("AppendOverallFeedback(%q) err = %v", reason, err)
		}
	}

	got, err := m.ExportFeedback(t.Context(), userID)
	if err != nil {
		t.Fatalf("ExportFeedback() err = %v", err)
	}

	const golden = "testdata/feedback_pack.golden.md"
	if *update {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(want), got); diff != "" {
		t.Errorf("ExportFeedback() mismatch (-want +got):\n%s", diff)
	}
}

func TestClearFeedback(t *testing.T) {
	const userID = "user-1"
	st := newFakeStore()
	m := scoring.Build(newDeps(t, st))
	svc := scoring.NewService(newDeps(t, st))
	for range 2 {
		if _, err := svc.AppendOverallFeedback(t.Context(), userID, dto.OverallFeedbackInput{Reason: "r"}); err != nil {
			t.Fatalf("AppendOverallFeedback() err = %v", err)
		}
	}

	n, err := m.ClearFeedback(t.Context(), userID)
	if err != nil || n != 2 {
		t.Fatalf("ClearFeedback() = %d, %v, want 2, nil", n, err)
	}
	if got, _ := svc.ListFeedback(t.Context(), userID, dto.ScoreFeedbackQuery{}); len(got.Entries) != 0 {
		t.Errorf("ListFeedback() after clear = %+v, want empty", got)
	}
}

func TestCompanyProfiles(t *testing.T) {
	const userID = "user-1"
	st := newFakeStore()
	st.SeedOptions(append([]dto.ScoringOption{retiredCobol}, bank...))
	st.SeedSearchConfig(picking(userID, "tech:go", "tech:rust", "tech:cobol"))

	yes := dto.Answer{PYes: 0.9, PNo: 0.05, PNotStated: 0.05}
	no := dto.Answer{PYes: 0.05, PNo: 0.9, PNotStated: 0.05}
	unsure := dto.Answer{PYes: 0.4, PNo: 0.3, PNotStated: 0.3}
	goHash, rustHash := scoring.QuestionHash("Does the role use Go?"), scoring.QuestionHash("Does the role use Rust?")
	for _, j := range []struct {
		id      string
		company string
		answers map[string]dto.Answer
	}{
		{"a1", "acme", map[string]dto.Answer{goHash: yes, rustHash: unsure}},
		{"a2", "acme", map[string]dto.Answer{goHash: no}},
		{"a3", "acme", nil},
		{"b1", "bare", nil},
	} {
		st.SeedJob(dto.Job{ID: j.id, CompanyID: j.company, ContentFingerprint: "fp"}, nil)
		st.SeedAnswers(j.id, "fp", jev.Model, j.answers)
	}

	m := scoring.Build(scoring.Deps{Store: st})
	got, err := m.CompanyProfiles(t.Context(), userID, []string{"acme", "bare", "empty"})
	if err != nil {
		t.Fatalf("CompanyProfiles() err = %v", err)
	}

	want := map[string][]dto.CompanyProfileEntry{
		"acme": {
			{Dimension: dto.DimensionTech, Label: "Go", Yes: 1, Known: 2, Total: 3},
			{Dimension: dto.DimensionTech, Label: "Rust", Total: 3},
		},
		"bare": {
			{Dimension: dto.DimensionTech, Label: "Go", Total: 1},
			{Dimension: dto.DimensionTech, Label: "Rust", Total: 1},
		},
		"empty": {
			{Dimension: dto.DimensionTech, Label: "Go"},
			{Dimension: dto.DimensionTech, Label: "Rust"},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("CompanyProfiles() mismatch (-want +got):\n%s", diff)
	}
}

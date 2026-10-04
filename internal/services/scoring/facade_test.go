package scoring_test

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/store"
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

	got, err := m.ExportFeedback(t.Context(), userID, false)
	if err != nil {
		t.Fatalf("ExportFeedback() err = %v", err)
	}
	assertGolden(t, "testdata/feedback_pack.golden.md", got)
}

func TestExportFeedback_Outdated(t *testing.T) {
	const userID = "user-1"
	st := newFakeStore()
	seedScoredJob(st, userID, "tech:go", "tech:cobol")
	svc := scoring.NewService(newDeps(t, st))
	if _, err := svc.AppendJobFeedback(t.Context(), userID, dto.JobFeedbackInput{JobID: testJob.ID, Direction: "lower", Reason: "Go is a given here."}); err != nil {
		t.Fatalf("AppendJobFeedback() err = %v", err)
	}
	if _, err := svc.AppendOverallFeedback(t.Context(), userID, dto.OverallFeedbackInput{Reason: "Scores run hot."}); err != nil {
		t.Fatalf("AppendOverallFeedback() err = %v", err)
	}
	st.SeedSearchConfig(picking(userID, "tech:go"))
	if _, err := svc.AppendOverallFeedback(t.Context(), userID, dto.OverallFeedbackInput{Reason: "Fine after the Pick change."}); err != nil {
		t.Fatalf("AppendOverallFeedback() err = %v", err)
	}
	m := scoring.Build(newDeps(t, st))

	omitted, err := m.ExportFeedback(t.Context(), userID, false)
	if err != nil {
		t.Fatalf("ExportFeedback(false) err = %v", err)
	}
	if !strings.Contains(omitted, "1 overall · 0 Job entries · 0 Collection entries · 2 outdated omitted") || strings.Contains(omitted, "Scores run hot.") {
		t.Errorf("ExportFeedback(false) = %q, want the header counting 2 outdated omitted and no outdated entry", omitted)
	}

	included, err := m.ExportFeedback(t.Context(), userID, true)
	if err != nil {
		t.Fatalf("ExportFeedback(true) err = %v", err)
	}
	assertGolden(t, "testdata/feedback_pack_outdated.golden.md", included)
}

func assertGolden(t *testing.T, golden, got string) {
	t.Helper()
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
		t.Errorf("%s mismatch (-want +got):\n%s", golden, diff)
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

func replayFixture(t *testing.T, negatives int) *scoring.Module {
	t.Helper()
	const userID = "user-1"
	st := newFakeStore()
	st.SeedSearchConfig(picking(userID, "tech:go", "tech:rust"))
	goHash, rustHash := scoring.QuestionHash("Does the role use Go?"), scoring.QuestionHash("Does the role use Rust?")
	yes := dto.Answer{PYes: 0.9, PNo: 0.05, PNotStated: 0.05}
	no := dto.Answer{PYes: 0.05, PNo: 0.9, PNotStated: 0.05}
	answers := []map[string]dto.Answer{{goHash: yes, rustHash: yes}, {goHash: yes}, {goHash: no, rustHash: no}, nil}

	var inputs []store.ScoringInput
	add := func(id, title string, a map[string]dto.Answer) {
		job := dto.Job{ID: id, Title: title, CompanySlug: "acme", ContentFingerprint: "fp"}
		st.SeedJob(job, nil)
		inputs = append(inputs, store.ScoringInput{Job: job, Answers: a})
	}
	add("great", "Platform Engineer", answers[0])
	add("applied", "Go Developer", answers[1])
	add("low", "COBOL | Maintainer", answers[2])
	add("plain", "Support Engineer", answers[3])
	for i := range negatives {
		add(fmt.Sprintf("no-%d", i), fmt.Sprintf("Dismissed %d", i), answers[i%len(answers)])
	}
	st.SeedScoringInputs(userID, inputs)

	svc := scoring.NewService(newDeps(t, st))
	grade := func(id, g string, reasons ...string) {
		if _, err := svc.SetGrade(t.Context(), userID, dto.GradeInput{JobID: id, Grade: g, Reasons: reasons}); err != nil {
			t.Fatalf("SetGrade(%s) err = %v", id, err)
		}
	}
	if negatives == 0 {
		st.SeedImpliedPositives(userID, dto.ImpliedLabel{JobID: "applied", Source: "application"}, dto.ImpliedLabel{JobID: "low", Source: "kept_cv"})
	} else {
		grade("great", "great", "tech")
		grade("applied", "ok")
		grade("low", "ok")
		reasons := [][]string{{"culture"}, {"tech", "culture"}, {"salary"}}
		for i := range negatives {
			grade(fmt.Sprintf("no-%d", i), "no", reasons[i%len(reasons)]...)
		}
		st.SeedImpliedPositives(userID, dto.ImpliedLabel{JobID: "no-0", Source: "application"})
	}
	return scoring.Build(newDeps(t, st))
}

func TestReplay(t *testing.T) {
	t.Run("positives only hides concordance", func(t *testing.T) {
		got, err := replayFixture(t, 0).Replay(t.Context(), "user-1")
		if err != nil {
			t.Fatalf("Replay() err = %v", err)
		}
		if strings.Contains(got, "Concordance") {
			t.Errorf("Replay() = %q, want no Concordance line", got)
		}
		assertGolden(t, "testdata/replay_positives_only.golden.md", got)
	})

	t.Run("five negatives show concordance, and a Grade outranks an application", func(t *testing.T) {
		got, err := replayFixture(t, 5).Replay(t.Context(), "user-1")
		if err != nil {
			t.Fatalf("Replay() err = %v", err)
		}
		if !strings.Contains(got, "Concordance") {
			t.Errorf("Replay() = %q, want a Concordance line", got)
		}
		if !strings.Contains(got, "| culture | 4 | 0 |") {
			t.Errorf("Replay() = %q, want 4 negatives counted under culture", got)
		}
		assertGolden(t, "testdata/replay.golden.md", got)
	})
}

func TestReplay_UnscoredLabel(t *testing.T) {
	st := newFakeStore()
	st.SeedImpliedPositives("user-1", dto.ImpliedLabel{JobID: "never-scored", Source: "application"})
	got, err := scoring.Build(newDeps(t, st)).Replay(t.Context(), "user-1")
	if err != nil {
		t.Fatalf("Replay() err = %v", err)
	}
	if !strings.Contains(got, "1 labelled jobs without a score excluded") {
		t.Errorf("Replay() = %q, want the unscored label counted", got)
	}
}

func TestFavouriteScoresIdenticallyOnEveryPath(t *testing.T) {
	const userID = "user-1"
	goKey := scoring.QuestionHash("Does the role use Go?")
	answers := map[string]dto.Answer{goKey: {PYes: 0.9, PNo: 0.05, PNotStated: 0.05}}
	job := dto.Job{ID: "job-1", Title: "Backend Engineer", CompanySlug: "acme", ContentFingerprint: "fp-1"}
	const unstarred = 37

	effect := func(t *testing.T) int {
		t.Helper()
		st := newFakeStore()
		st.SeedAnswers(job.ID, job.ContentFingerprint, jev.Model, answers)
		cfg := picking(userID, "tech:go")
		cfg.CompanyIsFavourite = true
		st.SeedJob(job, []dto.SearchConfig{cfg})
		st.SeedEffect(dto.AnswerEffect{ID: "effect-1", JobID: job.ID, Fingerprint: job.ContentFingerprint, Attempts: 1})
		runTick(t, st)
		return st.Completed()[0].Scores[0].Score
	}
	recompute := func(t *testing.T) int {
		t.Helper()
		st := newFakeStore()
		st.SeedSearchConfig(picking(userID, "tech:go"))
		st.SeedScoringInputs(userID, []store.ScoringInput{{Job: job, Answers: answers, Favourite: true}})
		if _, err := newService(t, st).Recompute(t.Context(), userID); err != nil {
			t.Fatalf("Recompute err = %v", err)
		}
		return st.Recomputed()[0].Score
	}
	port := func(t *testing.T) int {
		t.Helper()
		st := newFakeStore()
		st.SeedSearchConfig(picking(userID, "tech:go"))
		st.SeedScoringInputs(userID, []store.ScoringInput{{Job: job, Answers: answers, Favourite: true}})
		if err := scoring.Build(newDeps(t, st)).CompanyFavouriteChanged(t.Context(), nil, userID, ""); err != nil {
			t.Fatalf("CompanyFavouriteChanged err = %v", err)
		}
		return st.Recomputed()[0].Score
	}

	want := effect(t)
	if want <= unstarred {
		t.Fatalf("favourite score = %d, want above %d", want, unstarred)
	}
	for name, path := range map[string]func(*testing.T) int{"recompute": recompute, "port": port} {
		if got := path(t); got != want {
			t.Errorf("%s score = %d, want %d (same as the answer effect)", name, got, want)
		}
	}

	t.Run("replay", func(t *testing.T) {
		st := newFakeStore()
		st.SeedSearchConfig(picking(userID, "tech:go"))
		st.SeedScoringInputs(userID, []store.ScoringInput{{Job: job, Answers: answers, Favourite: true}})
		st.SeedImpliedPositives(userID, dto.ImpliedLabel{JobID: job.ID, Source: "application"})
		got, err := scoring.Build(newDeps(t, st)).Replay(t.Context(), userID)
		if err != nil {
			t.Fatalf("Replay err = %v", err)
		}
		if wantCell := fmt.Sprintf("| %d |", want); !strings.Contains(got, wantCell) {
			t.Errorf("Replay() = %q, want a row scored %q", got, wantCell)
		}
	})
}

func TestCorrectionRescoreReadsFavourite(t *testing.T) {
	const userID = "user-1"
	score := func(t *testing.T, starred bool) int {
		t.Helper()
		st := newFakeStore()
		seedScoredJob(st, userID, "tech:go")
		if starred {
			st.SeedFavouriteJob(userID, testJob.ID)
		}
		got, err := newService(t, st).SetCorrection(t.Context(), userID, dto.CorrectionInput{JobID: testJob.ID, OptionID: "tech:go", Value: "yes"})
		if err != nil {
			t.Fatalf("SetCorrection err = %v", err)
		}
		return got.Score
	}

	if starred, plain := score(t, true), score(t, false); starred <= plain {
		t.Errorf("starred score = %d, want above the unstarred %d", starred, plain)
	}
}

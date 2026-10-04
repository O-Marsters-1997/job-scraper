package scoring_test

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/scoringtest"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/store"
)

const fitUser = "user-1"

func fitFixture(t *testing.T, jobs int, params *dto.ScoringParams) *scoringtest.FakeStore {
	t.Helper()
	st := newFakeStore()
	cfg := picking(fitUser, "tech:go", "role:backend", "domain:gambling")
	cfg.Preferences.Scoring = params
	st.SeedSearchConfig(cfg)
	hashes := []string{
		scoring.QuestionHash("Does the role use Go?"),
		scoring.QuestionHash("Is this primarily a backend role?"),
		scoring.QuestionHash("Is the company's main business gambling?"),
	}
	rng := rand.New(rand.NewPCG(7, 8))
	inputs := make([]store.ScoringInput, jobs)
	for i := range inputs {
		answers := make(map[string]dto.Answer, len(hashes))
		for _, h := range hashes {
			answers[h] = dto.Answer{PNo: 1}
			if rng.Float64() < 0.5 {
				answers[h] = dto.Answer{PYes: 1}
			}
		}
		job := dto.Job{ID: fmt.Sprintf("job-%d", i), Title: "Engineer", CompanySlug: "acme", ContentFingerprint: "fp"}
		st.SeedJob(job, nil)
		inputs[i] = store.ScoringInput{Job: job, Answers: answers}
	}
	st.SeedScoringInputs(fitUser, inputs)
	return st
}

func gradeJobs(t *testing.T, svc *scoring.Service, from, to int, grade func(i int) string) {
	t.Helper()
	for i := from; i < to; i++ {
		if _, err := svc.SetGrade(t.Context(), fitUser, dto.GradeInput{JobID: fmt.Sprintf("job-%d", i), Grade: grade(i)}); err != nil {
			t.Fatalf("SetGrade(job-%d) err = %v", i, err)
		}
	}
}

func alternating(i int) string {
	if i%2 == 0 {
		return "great"
	}
	return "no"
}

func TestFit(t *testing.T) {
	t.Run("fewer than 30 grades calibrates nothing", func(t *testing.T) {
		st := fitFixture(t, 40, nil)
		svc := newService(t, st)
		gradeJobs(t, svc, 0, 29, alternating)

		report, err := svc.Fit(t.Context(), fitUser)
		if err != nil {
			t.Fatalf("Fit() err = %v", err)
		}

		cfg, _ := st.GetSearchConfig(t.Context(), fitUser)
		if cfg.Preferences.Scoring != nil {
			t.Errorf("Scoring = %+v, want nil so the defaults apply", cfg.Preferences.Scoring)
		}
		if !strings.Contains(report, "defaults apply") {
			t.Errorf("Fit() = %q, want it to say the defaults apply", report)
		}
	})

	t.Run("a challenger that loses cross-validation is rejected and weights stay", func(t *testing.T) {
		champion := map[dto.Dimension]float64{dto.DimensionRole: 0.2, dto.DimensionTech: 1, dto.DimensionDomain: 6}
		st := fitFixture(t, 80, &dto.ScoringParams{Weights: champion})
		svc := newService(t, st)
		scores := championScores(t, st)
		for i := range scores {
			grade := "no"
			if (scores[i] >= medianScore(scores)) != (i%7 == 0) {
				grade = "great"
			}
			if _, err := st.UpsertGrade(t.Context(), fitUser, dto.Grade{JobID: fmt.Sprintf("job-%d", i), Grade: grade}); err != nil {
				t.Fatalf("UpsertGrade err = %v", err)
			}
		}

		report, err := svc.Fit(t.Context(), fitUser)
		if err != nil {
			t.Fatalf("Fit() err = %v", err)
		}

		cfg, _ := st.GetSearchConfig(t.Context(), fitUser)
		got := cfg.Preferences.Scoring
		if !strings.Contains(report, "Weights rejected") {
			t.Fatalf("Fit() = %q, want the challenger rejected", report)
		}
		for dim, want := range champion {
			if got.Weights[dim] != want {
				t.Errorf("stored weight %s = %v, want %v unchanged", dim, got.Weights[dim], want)
			}
		}
	})

	t.Run("grading refits at 30 and again only after 15 more", func(t *testing.T) {
		st := fitFixture(t, 60, nil)
		svc := newService(t, st)
		fittedAt := func() int {
			cfg, _ := st.GetSearchConfig(t.Context(), fitUser)
			if cfg.Preferences.Scoring == nil {
				return 0
			}
			return cfg.Preferences.Scoring.GradeCount
		}

		gradeJobs(t, svc, 0, 30, alternating)
		if got := fittedAt(); got != 30 {
			t.Fatalf("GradeCount after 30 grades = %d, want 30", got)
		}
		gradeJobs(t, svc, 30, 44, alternating)
		if got := fittedAt(); got != 30 {
			t.Errorf("GradeCount after 44 grades = %d, want 30 (no refit yet)", got)
		}
		gradeJobs(t, svc, 44, 45, alternating)
		if got := fittedAt(); got != 45 {
			t.Errorf("GradeCount after 45 grades = %d, want 45", got)
		}
	})

	t.Run("replay reports the weights in use", func(t *testing.T) {
		st := fitFixture(t, 3, &dto.ScoringParams{Weights: map[dto.Dimension]float64{dto.DimensionTech: 5}, GradeCount: 30})
		got, err := scoring.Build(newDeps(t, st)).Replay(t.Context(), fitUser)
		if err != nil {
			t.Fatalf("Replay() err = %v", err)
		}
		if !strings.Contains(got, "Weights: fitted: tech 5.00") {
			t.Errorf("Replay() = %q, want the fitted tech weight", got)
		}
	})
}

func championScores(t *testing.T, st *scoringtest.FakeStore) []int {
	t.Helper()
	if _, err := newService(t, st).Recompute(t.Context(), fitUser); err != nil {
		t.Fatalf("Recompute() err = %v", err)
	}
	byJob := make(map[string]int)
	for _, sc := range st.Recomputed() {
		byJob[sc.JobID] = sc.Score
	}
	scores := make([]int, len(byJob))
	for i := range scores {
		scores[i] = byJob[fmt.Sprintf("job-%d", i)]
	}
	return scores
}

func medianScore(scores []int) int {
	return slices.Sorted(slices.Values(scores))[len(scores)/2]
}

func TestUpdateConfigKeepsFittedParams(t *testing.T) {
	params := &dto.ScoringParams{Weights: map[dto.Dimension]float64{dto.DimensionTech: 5}, GradeCount: 30}
	st := fitFixture(t, 1, params)

	if _, err := newService(t, st).UpdateConfig(t.Context(), fitUser, dto.ScoringConfigView{NotifyThreshold: 60}); err != nil {
		t.Fatalf("UpdateConfig() err = %v", err)
	}

	cfg, _ := st.GetSearchConfig(t.Context(), fitUser)
	if cfg.Preferences.Scoring == nil || cfg.Preferences.Scoring.GradeCount != 30 {
		t.Errorf("Scoring after UpdateConfig = %+v, want the fitted params kept", cfg.Preferences.Scoring)
	}
}

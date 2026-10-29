package scoring_test

import (
	"fmt"
	"math/rand/v2"
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
	t.Run("is disabled and changes nothing", func(t *testing.T) {
		t.Parallel()
		params := &dto.ScoringParams{Weights: map[dto.Dimension]float64{dto.DimensionTech: 5}, GradeCount: 30}
		st := fitFixture(t, 60, params)
		svc := newService(t, st)
		gradeJobs(t, svc, 0, 60, alternating)

		report, err := svc.Fit(t.Context(), fitUser)

		if err == nil || !strings.Contains(err.Error(), "disabled") {
			t.Fatalf("Fit() = %q, %v, want the disabled error", report, err)
		}
		cfg, _ := st.GetSearchConfig(t.Context(), fitUser)
		if cfg.Preferences.Scoring == nil || cfg.Preferences.Scoring.GradeCount != 30 {
			t.Errorf("Scoring = %+v, want it untouched", cfg.Preferences.Scoring)
		}
		if got := len(st.Recomputed()); got != 0 {
			t.Errorf("Recomputed() = %d scores, want none", got)
		}
	})

	t.Run("grading never refits", func(t *testing.T) {
		t.Parallel()
		st := fitFixture(t, 60, nil)
		svc := newService(t, st)

		gradeJobs(t, svc, 0, 60, alternating)
		cfg, err := st.GetSearchConfig(t.Context(), fitUser)
		if err != nil {
			t.Fatalf("GetSearchConfig() err = %v", err)
		}
		if cfg.Preferences.Scoring != nil {
			t.Errorf("Scoring after 60 grades = %+v, want nil", cfg.Preferences.Scoring)
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

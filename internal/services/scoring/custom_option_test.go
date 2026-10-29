package scoring_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
)

type fakeDrafter struct {
	calls int
	dims  []dto.DimensionSpec
	out   dto.DraftedQuestion
	err   error
}

func (f *fakeDrafter) Draft(_ context.Context, _, _ string, dims []dto.DimensionSpec) (dto.DraftedQuestion, error) {
	f.calls++
	f.dims = dims
	return f.out, f.err
}

func withDrafter(d scoring.Drafter) depsOpt { return func(deps *scoring.Deps) { deps.Drafter = d } }

func TestService_DraftCustomOption(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	newSvc := func(t *testing.T, key string, drafter *fakeDrafter, jobCount int) *scoring.Service {
		t.Helper()
		st := newFakeStore()
		var jobs []dto.Job
		for i := range jobCount {
			job := dto.Job{ID: fmt.Sprintf("job-%d", i), Title: fmt.Sprintf("Engineer %d", i), CompanySlug: "acme", ContentFingerprint: "fp"}
			st.SeedJob(job, nil)
			jobs = append(jobs, job)
		}
		st.SeedRecentScoredJobs("user-1", jobs...)
		return newService(t, st, withDrafter(drafter), withKey(key))
	}

	t.Run("a phrase is drafted into a question answered on five recent jobs", func(t *testing.T) {
		t.Parallel()
		drafter := &fakeDrafter{out: dto.DraftedQuestion{Question: "Does it offer a 4-day week?", Dimension: dto.DimensionWork}}
		svc := newSvc(t, "sk-or-test", drafter, 7)

		got, err := svc.DraftCustomOption(ctx, "user-1", dto.CustomOptionDraftInput{Phrase: "4 day week"})
		if err != nil {
			t.Fatal(err)
		}

		if got.Question != "Does it offer a 4-day week?" || got.Dimension != dto.DimensionWork {
			t.Errorf("draft = %q in %q, want the drafter's question and dimension", got.Question, got.Dimension)
		}
		if len(got.Answers) != 5 {
			t.Fatalf("answers = %d, want 5", len(got.Answers))
		}
		want := dto.DraftAnswerOnJob{JobID: "job-0", Title: "Engineer 0", Company: "acme", Resolved: "yes"}
		if diff := cmp.Diff(want, got.Answers[0]); diff != "" {
			t.Errorf("first answer (-want +got):\n%s", diff)
		}
		for _, d := range drafter.dims {
			if d.Kind == "ladder" {
				t.Errorf("drafter offered ladder dimension %q", d.Key)
			}
		}
	})

	t.Run("an edited question is previewed without calling the drafter", func(t *testing.T) {
		t.Parallel()
		drafter := &fakeDrafter{}
		svc := newSvc(t, "sk-or-test", drafter, 2)

		got, err := svc.DraftCustomOption(ctx, "user-1", dto.CustomOptionDraftInput{Phrase: "4 day week", Question: "Is it a 4 day week?", Dimension: dto.DimensionWork})
		if err != nil {
			t.Fatal(err)
		}

		if drafter.calls != 0 {
			t.Errorf("drafter calls = %d, want 0", drafter.calls)
		}
		if got.Question != "Is it a 4 day week?" || got.Dimension != dto.DimensionWork || len(got.Answers) != 2 {
			t.Errorf("draft = %+v, want the given question on both jobs", got)
		}
	})

	t.Run("no credential is unprocessable", func(t *testing.T) {
		t.Parallel()
		drafter := &fakeDrafter{}
		svc := newSvc(t, "", drafter, 2)

		_, err := svc.DraftCustomOption(ctx, "user-1", dto.CustomOptionDraftInput{Phrase: "4 day week"})

		if status, _ := apperr.StatusFor(err); status != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d (err %v), want 422", status, err)
		}
		if drafter.calls != 0 {
			t.Errorf("drafter calls = %d, want 0", drafter.calls)
		}
	})

	t.Run("an edited question without a credential is unprocessable", func(t *testing.T) {
		t.Parallel()
		svc := newSvc(t, "", &fakeDrafter{}, 0)
		_, err := svc.DraftCustomOption(ctx, "user-1", dto.CustomOptionDraftInput{Phrase: "x", Question: "Is it remote?"})
		if status, _ := apperr.StatusFor(err); status != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d (err %v), want 422", status, err)
		}
	})

	t.Run("an edited question with a ladder dimension is invalid", func(t *testing.T) {
		t.Parallel()
		svc := newSvc(t, "sk-or-test", &fakeDrafter{}, 1)
		_, err := svc.DraftCustomOption(ctx, "user-1", dto.CustomOptionDraftInput{Phrase: "x", Question: "Is it senior?", Dimension: dto.DimensionSeniority})
		if status, _ := apperr.StatusFor(err); status != http.StatusBadRequest {
			t.Fatalf("status = %d (err %v), want 400", status, err)
		}
	})

	t.Run("blank phrase is invalid", func(t *testing.T) {
		t.Parallel()
		svc := newSvc(t, "sk-or-test", &fakeDrafter{}, 0)
		_, err := svc.DraftCustomOption(ctx, "user-1", dto.CustomOptionDraftInput{Phrase: "  "})
		if status, _ := apperr.StatusFor(err); status != http.StatusBadRequest {
			t.Fatalf("status = %d (err %v), want 400", status, err)
		}
	})

	t.Run("a ladder dimension from the drafter is rejected", func(t *testing.T) {
		t.Parallel()
		drafter := &fakeDrafter{out: dto.DraftedQuestion{Question: "Is it senior?", Dimension: dto.DimensionSeniority}}
		svc := newSvc(t, "sk-or-test", drafter, 1)
		if _, err := svc.DraftCustomOption(ctx, "user-1", dto.CustomOptionDraftInput{Phrase: "senior"}); err == nil {
			t.Fatal("err = nil, want a rejection")
		}
	})

	t.Run("drafter failure is returned", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("boom")
		svc := newSvc(t, "sk-or-test", &fakeDrafter{err: boom}, 1)
		if _, err := svc.DraftCustomOption(ctx, "user-1", dto.CustomOptionDraftInput{Phrase: "x"}); !errors.Is(err, boom) {
			t.Fatalf("err = %v, want boom", err)
		}
	})
}

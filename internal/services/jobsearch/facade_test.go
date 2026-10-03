package jobsearch_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/queue/queuetest"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
)

type recordingQueue struct {
	jobsearchtest.NoopQueue
	tasks   []queue.Task
	failFor string
}

func (q *recordingQueue) Publish(_ context.Context, task queue.Task) error {
	if task.BoardID == q.failFor {
		return errors.New("broker down")
	}
	q.tasks = append(q.tasks, task)
	return nil
}

func seedBoards(t *testing.T, st *jobsearchtest.FakeStore) (verified, candidate dto.CompanyBoard) {
	t.Helper()
	ctx := t.Context()
	company, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Name: "Acme", Slug: "acme"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = st.UpsertCandidateBoard(ctx, company.ID, "greenhouse", "acme"); err != nil {
		t.Fatal(err)
	}
	if verified, err = st.VerifyCompanyBoard(ctx, company.ID, "greenhouse", "acme", "user_confirmed", ""); err != nil {
		t.Fatal(err)
	}
	if candidate, err = st.UpsertCandidateBoard(ctx, company.ID, "lever", "acme"); err != nil {
		t.Fatal(err)
	}
	return verified, candidate
}

func TestPublishBoardChecks(t *testing.T) {
	ids := func(q *recordingQueue) []string {
		var out []string
		for _, task := range q.tasks {
			out = append(out, task.BoardID)
		}
		slices.Sort(out)
		return out
	}

	t.Run("scheduled run publishes only due Boards", func(t *testing.T) {
		st, q := jobsearchtest.NewFakeStore(), &recordingQueue{}
		verified, _ := seedBoards(t, st)
		m := jobsearch.Build(jobsearch.Deps{Store: st, Scoring: jobsearchtest.NewNoopScoring(), Queue: q})

		if err := m.PublishBoardChecks(t.Context(), false); err != nil {
			t.Fatal(err)
		}
		if got := ids(q); !slices.Equal(got, []string{verified.ID}) || q.tasks[0].Manual {
			t.Errorf("tasks = %+v, want one non-manual task for %s", q.tasks, verified.ID)
		}
	})

	t.Run("manual run publishes every active Board", func(t *testing.T) {
		st, q := jobsearchtest.NewFakeStore(), &recordingQueue{}
		verified, candidate := seedBoards(t, st)
		m := jobsearch.Build(jobsearch.Deps{Store: st, Scoring: jobsearchtest.NewNoopScoring(), Queue: q})

		if err := m.PublishBoardChecks(t.Context(), true); err != nil {
			t.Fatal(err)
		}
		want := []string{verified.ID, candidate.ID}
		slices.Sort(want)
		if got := ids(q); !slices.Equal(got, want) {
			t.Fatalf("published = %v, want %v", got, want)
		}
		for _, task := range q.tasks {
			if !task.Manual || task.Kind != queue.BoardCheckTask {
				t.Errorf("task = %+v, want manual Board check", task)
			}
		}
	})

	t.Run("one failed publish does not stop the rest", func(t *testing.T) {
		st, q := jobsearchtest.NewFakeStore(), &recordingQueue{}
		verified, candidate := seedBoards(t, st)
		q.failFor = min(verified.ID, candidate.ID)
		m := jobsearch.Build(jobsearch.Deps{Store: st, Scoring: jobsearchtest.NewNoopScoring(), Queue: q})

		if err := m.PublishBoardChecks(t.Context(), true); err != nil {
			t.Fatal(err)
		}
		if len(q.tasks) != 1 || q.tasks[0].BoardID == q.failFor {
			t.Errorf("tasks = %+v, want only the Board that did not fail", q.tasks)
		}
	})
}

type claimTaken struct{ *jobsearchtest.FakeStore }

func (claimTaken) ClaimRecoverableSourceTarget(context.Context, string, string) (dto.SourceTarget, error) {
	return dto.SourceTarget{}, data.ErrNotFound
}

func startRun(t *testing.T, st *jobsearchtest.FakeStore, source, value string) dto.SourceTarget {
	t.Helper()
	target, err := st.CreateSourceTargetWithRun(t.Context(), "user-1", source, value, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	target, err = st.TransitionSourceTargetRun(t.Context(), target.ID, target.RunID, "running", "")
	if err != nil {
		t.Fatal(err)
	}
	return target
}

func newRecoverModule(t *testing.T, st jobsearchtest.Store) (*jobsearch.Module, *queuetest.Recorder) {
	t.Helper()
	rec := queuetest.NewRecorder()
	deps := jobsearchtest.NewDeps(st)
	deps.Queue = rec
	return jobsearch.Build(deps), rec
}

func TestRecoverRuns(t *testing.T) {
	ctx := t.Context()
	ignoreID := cmpopts.IgnoreFields(queue.Task{}, "ID")

	t.Run("stuck ATS target republishes a Board check for its verified Board", func(t *testing.T) {
		st := jobsearchtest.NewFakeStore()
		company, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := st.UpsertCandidateBoard(ctx, company.ID, "greenhouse", "acme"); err != nil {
			t.Fatal(err)
		}
		board, err := st.VerifyCompanyBoard(ctx, company.ID, "greenhouse", "acme", "test", "")
		if err != nil {
			t.Fatal(err)
		}
		target := startRun(t, st, "greenhouse", "acme")
		m, rec := newRecoverModule(t, st)

		if err := m.RecoverRuns(ctx); err != nil {
			t.Fatalf("RecoverRuns() = %v, want nil", err)
		}

		want := []queue.Task{{
			Version: 1, Source: "greenhouse", Kind: queue.BoardCheckTask, TargetID: target.ID,
			RunID: target.RunID, BoardID: board.ID, Manual: true, Recovery: true,
		}}
		if diff := cmp.Diff(want, rec.Tasks(), ignoreID); diff != "" {
			t.Fatalf("published tasks mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("stuck discovery target republishes a listing page", func(t *testing.T) {
		st := jobsearchtest.NewFakeStore()
		target := startRun(t, st, "linkedin", "go")
		m, rec := newRecoverModule(t, st)

		if err := m.RecoverRuns(ctx); err != nil {
			t.Fatalf("RecoverRuns() = %v, want nil", err)
		}

		want := []queue.Task{{
			Version: 1, Source: "linkedin", Kind: queue.ListingPageTask, TargetID: target.ID,
			RunID: target.RunID, Recovery: true,
		}}
		if diff := cmp.Diff(want, rec.Tasks(), ignoreID); diff != "" {
			t.Fatalf("published tasks mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("target claimed elsewhere is skipped without error", func(t *testing.T) {
		st := jobsearchtest.NewFakeStore()
		startRun(t, st, "linkedin", "go")
		m, rec := newRecoverModule(t, claimTaken{st})

		if err := m.RecoverRuns(ctx); err != nil {
			t.Fatalf("RecoverRuns() = %v, want nil", err)
		}
		if got := rec.Tasks(); len(got) != 0 {
			t.Fatalf("published %d tasks, want 0", len(got))
		}
	})
}

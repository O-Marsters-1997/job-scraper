package jobsearch_test

import (
	"context"
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
		board, err := st.VerifyCompanyBoard(ctx, company.ID, "greenhouse", "acme", "test")
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

package jobsearch_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
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
	ctx := context.Background()
	company, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Name: "Acme", Slug: "acme"})
	if err != nil {
		t.Fatal(err)
	}
	verified, err = st.UpsertCandidateBoard(ctx, company.ID, "greenhouse", "acme")
	if err != nil {
		t.Fatal(err)
	}
	if verified, err = st.VerifyCompanyBoard(ctx, company.ID, "greenhouse", "acme", "user_confirmed"); err != nil {
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

		if err := m.PublishBoardChecks(context.Background(), false); err != nil {
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

		if err := m.PublishBoardChecks(context.Background(), true); err != nil {
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

		if err := m.PublishBoardChecks(context.Background(), true); err != nil {
			t.Fatal(err)
		}
		if len(q.tasks) != 1 || q.tasks[0].BoardID == q.failFor {
			t.Errorf("tasks = %+v, want only the Board that did not fail", q.tasks)
		}
	})
}

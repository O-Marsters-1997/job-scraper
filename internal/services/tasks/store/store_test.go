package store_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/services/tasks/store"
	"github.com/ollymarsters/job-scraper/internal/services/tasks/taskstest"
)

func newStore(t *testing.T) (st *store.Store, pool *pgxpool.Pool, userID string) {
	t.Helper()
	pool = pgtest.New(t)
	return store.New(pool, nil), pool, pgtest.InsertUser(t, pool)
}

func insertType(t *testing.T, pool *pgxpool.Pool, userID, name string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(t.Context(),
		`INSERT INTO task_types (user_id, name, colour) VALUES ($1, $2, '#6366f1') RETURNING id`, userID, name).Scan(&id)
	if err != nil {
		t.Fatalf("insert task type: %v", err)
	}
	return id
}

func TestStoreContract(t *testing.T) {
	t.Parallel()
	taskstest.RunStoreContract(t, func(t *testing.T) taskstest.Fixture {
		t.Helper()
		st, pool, userID := newStore(t)
		other := pgtest.InsertUser(t, pool)
		return taskstest.Fixture{
			Store:       st,
			UserID:      userID,
			OtherUserID: other,
			TypeID:      insertType(t, pool, userID, "Apply"),
			OtherTypeID: insertType(t, pool, other, "Apply"),
			JobID:       pgtest.InsertJob(t, pool, "Contract Job", "Contract Job"),
		}
	})
}

func TestSeedDefaultTaskTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		commit bool
		want   []string
	}{
		{name: "commit keeps the defaults", commit: true, want: []string{"Apply", "Follow up", "Prep", "Reach out"}},
		{name: "rollback discards the defaults", commit: false, want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			st, pool, userID := newStore(t)
			pgtest.InTx(t, pool, tt.commit, func(tx pgx.Tx) error {
				return st.SeedDefaultTaskTypes(t.Context(), tx, userID)
			})

			types, err := st.ListTaskTypes(t.Context(), userID)
			if err != nil {
				t.Fatalf("ListTaskTypes err = %v", err)
			}
			got := make([]string, len(types))
			for i, ty := range types {
				got[i] = ty.Name
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("default task types mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreateTaskWithMissingJobReturnsNotFound(t *testing.T) {
	t.Parallel()
	st, pool, userID := newStore(t)
	typeID := insertType(t, pool, userID, "Apply")

	_, err := st.CreateTask(t.Context(), userID, dto.CreateTaskInput{TypeID: typeID, JobID: new("00000000-0000-0000-0000-000000000000")})
	if !errors.Is(err, data.ErrNotFound) {
		t.Errorf("CreateTask(missing job) err = %v, want ErrNotFound", err)
	}
}

func TestCreateTaskRejectsNonHTTPLink(t *testing.T) {
	t.Parallel()
	st, pool, userID := newStore(t)
	typeID := insertType(t, pool, userID, "Apply")

	if _, err := st.CreateTask(t.Context(), userID, dto.CreateTaskInput{TypeID: typeID, Link: "javascript:alert(1)"}); err == nil {
		t.Error("CreateTask(javascript: link) err = nil, want the CHECK constraint to reject it")
	}
}

func TestListTasksJoinsJobAndSortsByJobPosted(t *testing.T) {
	t.Parallel()
	st, pool, userID := newStore(t)
	typeID := insertType(t, pool, userID, "Apply")
	older := pgtest.InsertJob(t, pool, "Older", "Older")
	newer := pgtest.InsertJob(t, pool, "Newer", "Newer")
	postedNewer := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	for id, posted := range map[string]time.Time{older: postedNewer.AddDate(0, 0, -10), newer: postedNewer} {
		if _, err := pool.Exec(t.Context(), `UPDATE jobs SET posted_at = $2 WHERE id = $1`, id, posted); err != nil {
			t.Fatalf("set posted_at: %v", err)
		}
	}
	create := func(jobID *string) dto.Task {
		task, err := st.CreateTask(t.Context(), userID, dto.CreateTaskInput{TypeID: typeID, JobID: jobID})
		if err != nil {
			t.Fatalf("CreateTask err = %v", err)
		}
		return task
	}
	forOlder := create(&older)
	forNewer := create(&newer)
	noJob := create(nil)

	byPosted, err := st.ListTasks(t.Context(), userID, dto.TaskPageOptions{Sort: dto.TaskSortJobPosted, Limit: 25})
	if err != nil {
		t.Fatalf("ListTasks(job_posted) err = %v", err)
	}
	var got []string
	for _, task := range byPosted.Items {
		got = append(got, task.ID)
	}
	if diff := cmp.Diff([]string{forNewer.ID, forOlder.ID, noJob.ID}, got); diff != "" {
		t.Errorf("ListTasks(job_posted) ids mismatch (-want +got):\n%s", diff)
	}
	first := byPosted.Items[0]
	if first.JobTitle != "Newer" || first.JobCompanySlug != "acme" || first.JobPostedAt == nil || !first.JobPostedAt.Equal(postedNewer) {
		t.Errorf("ListTasks(job_posted)[0] job = %q/%q/%v, want Newer/acme/%v", first.JobTitle, first.JobCompanySlug, first.JobPostedAt, postedNewer)
	}
}

func TestListTasksCarriesApplicationID(t *testing.T) {
	t.Parallel()
	st, pool, userID := newStore(t)
	typeID := insertType(t, pool, userID, "Apply")
	jobID := pgtest.InsertJob(t, pool, "Role", "Role")
	if _, err := st.CreateTask(t.Context(), userID, dto.CreateTaskInput{TypeID: typeID, JobID: &jobID}); err != nil {
		t.Fatalf("CreateTask err = %v", err)
	}
	list := func() dto.Task {
		page, err := st.ListTasks(t.Context(), userID, dto.TaskPageOptions{Limit: 25})
		if err != nil || len(page.Items) != 1 {
			t.Fatalf("ListTasks = %d items, err %v, want 1 item", len(page.Items), err)
		}
		return page.Items[0]
	}
	if got := list().ApplicationID; got != nil {
		t.Errorf("ApplicationID before tracking = %v, want nil", *got)
	}

	var applicationID string
	err := pool.QueryRow(t.Context(),
		`INSERT INTO applications (user_id, job_id) VALUES ($1, $2) RETURNING id`, userID, jobID).Scan(&applicationID)
	if err != nil {
		t.Fatalf("insert application: %v", err)
	}
	if got := list().ApplicationID; got == nil || *got != applicationID {
		t.Errorf("ApplicationID after tracking = %v, want %q", got, applicationID)
	}
}

func TestDeletingTheJobKeepsTheTask(t *testing.T) {
	t.Parallel()
	st, pool, userID := newStore(t)
	typeID := insertType(t, pool, userID, "Apply")
	jobID := pgtest.InsertJob(t, pool, "Gone", "Gone")
	if _, err := st.CreateTask(t.Context(), userID, dto.CreateTaskInput{TypeID: typeID, JobID: &jobID}); err != nil {
		t.Fatalf("CreateTask err = %v", err)
	}
	if _, err := pool.Exec(t.Context(), `DELETE FROM jobs WHERE id = $1`, jobID); err != nil {
		t.Fatalf("delete job: %v", err)
	}

	page, err := st.ListTasks(t.Context(), userID, dto.TaskPageOptions{Sort: dto.TaskSortCreated, Limit: 25})
	if err != nil {
		t.Fatalf("ListTasks err = %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].JobID != nil {
		t.Errorf("ListTasks after job delete = %+v, want one task with no job", page.Items)
	}
}

package store_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/services/applications/applicationstest"
	"github.com/ollymarsters/job-scraper/internal/services/applications/store"
	"github.com/ollymarsters/job-scraper/internal/services/events"
	"github.com/ollymarsters/job-scraper/internal/services/events/eventstest"
	"github.com/ollymarsters/job-scraper/internal/services/tasks"
)

func newStore(t *testing.T) (st *store.Store, pool *pgxpool.Pool, userID string) {
	t.Helper()
	pool = pgtest.New(t)
	return store.New(pool, events.New(pool, eventstest.Unscored()), tasks.NewFacade(pool)), pool, pgtest.InsertUser(t, pool)
}

func TestStoreContract(t *testing.T) {
	t.Parallel()
	applicationstest.RunStoreContract(t, func(t *testing.T) applicationstest.Fixture {
		t.Helper()
		st, pool, userID := newStore(t)
		return applicationstest.Fixture{
			Store:       st,
			UserID:      userID,
			OtherUserID: pgtest.InsertUser(t, pool),
			JobID:       pgtest.InsertJob(t, pool, "Contract Job", "Contract Job"),
			NewJob:      func() string { return pgtest.InsertJob(t, pool, "Other Job", "Other Job") },
		}
	})
}

func TestCreateApplicationRejectsDuplicate(t *testing.T) {
	t.Parallel()
	st, pool, userID := newStore(t)
	jobID := pgtest.InsertJob(t, pool, "Engineer", "Engineer")

	if _, err := st.CreateApplication(t.Context(), userID, dto.CreateApplicationInput{JobID: jobID}); err != nil {
		t.Fatalf("first CreateApplication err = %v", err)
	}
	_, err := st.CreateApplication(t.Context(), userID, dto.CreateApplicationInput{JobID: jobID})
	if !errors.Is(err, store.ErrApplicationExists) {
		t.Errorf("duplicate CreateApplication err = %v, want ErrApplicationExists", err)
	}
}

func TestListApplicationsJoinsJobDetails(t *testing.T) {
	t.Parallel()
	st, pool, userID := newStore(t)
	jobID := pgtest.InsertJob(t, pool, "Staff Engineer", "Staff Engineer")
	if _, err := st.CreateApplication(t.Context(), userID, dto.CreateApplicationInput{JobID: jobID}); err != nil {
		t.Fatalf("CreateApplication err = %v", err)
	}

	got, err := st.ListApplications(t.Context(), userID, dto.ApplicationsQuery{})
	if err != nil {
		t.Fatalf("ListApplications err = %v", err)
	}
	if len(got) != 1 || got[0].JobTitle != "Staff Engineer" || got[0].JobID != jobID {
		t.Errorf("ListApplications = %+v, want one application for %q", got, "Staff Engineer")
	}
}

func TestListApplicationsFiltersByStatus(t *testing.T) {
	t.Parallel()
	st, pool, userID := newStore(t)
	matching := pgtest.InsertJob(t, pool, "Match", "Match")
	other := pgtest.InsertJob(t, pool, "Other", "Other")
	status, err := st.CreateApplicationStatus(t.Context(), userID, "Applied", "#6366f1", nil)
	if err != nil {
		t.Fatalf("CreateApplicationStatus err = %v", err)
	}
	for _, in := range []dto.CreateApplicationInput{
		{JobID: matching, StatusID: status.ID},
		{JobID: other},
	} {
		if _, err := st.CreateApplication(t.Context(), userID, in); err != nil {
			t.Fatalf("CreateApplication(%+v) err = %v", in, err)
		}
	}

	got, err := st.ListApplications(t.Context(), userID, dto.ApplicationsQuery{StatusID: status.ID})
	if err != nil {
		t.Fatalf("ListApplications err = %v", err)
	}
	if len(got) != 1 || got[0].JobID != matching {
		t.Errorf("ListApplications(status) = %+v, want only the job %s", got, matching)
	}
}

func TestListApplicationsReportsClosedJob(t *testing.T) {
	t.Parallel()
	st, pool, userID := newStore(t)
	closed := pgtest.InsertJob(t, pool, "Closed", "Closed")
	open := pgtest.InsertJob(t, pool, "Open", "Open")
	if _, err := pool.Exec(t.Context(), `UPDATE jobs SET closed_at = NOW() WHERE id = $1`, closed); err != nil {
		t.Fatalf("close job: %v", err)
	}
	for _, jobID := range []string{closed, open} {
		if _, err := st.CreateApplication(t.Context(), userID, dto.CreateApplicationInput{JobID: jobID}); err != nil {
			t.Fatalf("CreateApplication(%s) err = %v", jobID, err)
		}
	}

	got, err := st.ListApplications(t.Context(), userID, dto.ApplicationsQuery{})
	if err != nil {
		t.Fatalf("ListApplications err = %v", err)
	}
	for _, a := range got {
		if gotClosed, wantClosed := a.JobClosedAt != nil, a.JobID == closed; gotClosed != wantClosed {
			t.Errorf("ListApplications job %s closed = %v, want %v", a.JobID, gotClosed, wantClosed)
		}
	}
}

func TestSeedDefaultStatuses(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		commit bool
		want   int
	}{
		{name: "commit keeps the defaults", commit: true, want: 5},
		{name: "rollback discards the defaults", commit: false, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			st, pool, userID := newStore(t)
			pgtest.InTx(t, pool, tt.commit, func(tx pgx.Tx) error {
				return st.SeedDefaultStatuses(t.Context(), tx, userID)
			})

			got, err := st.ListApplicationStatusesByUser(t.Context(), userID)
			if err != nil {
				t.Fatalf("ListApplicationStatusesByUser err = %v", err)
			}
			if len(got) != tt.want {
				t.Errorf("default statuses = %d, want %d", len(got), tt.want)
			}
		})
	}
}

func TestSeededSystemStatuses(t *testing.T) {
	t.Parallel()
	st, pool, userID := newStore(t)
	jobID := pgtest.InsertJob(t, pool, "Engineer", "Engineer")
	pgtest.InTx(t, pool, true, func(tx pgx.Tx) error {
		return st.SeedDefaultStatuses(t.Context(), tx, userID)
	})

	all, err := st.ListApplicationStatusesByUser(t.Context(), userID)
	if err != nil {
		t.Fatalf("ListApplicationStatusesByUser err = %v", err)
	}
	keys := map[string]string{}
	for _, s := range all {
		if s.SystemKey != "" {
			keys[s.SystemKey] = s.Name
		}
	}
	if diff := cmp.Diff(map[string]string{"draft": "Draft", "applied": "Applied"}, keys); diff != "" {
		t.Errorf("seeded system keys mismatch (-want +got):\n%s", diff)
	}

	app, err := st.CreateApplication(t.Context(), userID, dto.CreateApplicationInput{JobID: jobID})
	if err != nil {
		t.Fatalf("CreateApplication err = %v", err)
	}
	var got string
	for _, s := range all {
		if s.ID == app.StatusID {
			got = s.Name
		}
	}
	if got != "Draft" {
		t.Errorf("CreateApplication(no status) status = %q, want Draft", got)
	}
}

func TestApplicationEventsShareTheApplicationsTransaction(t *testing.T) {
	t.Parallel()
	st, pool, userID := newStore(t)
	jobID := pgtest.InsertJob(t, pool, "Engineer", "Engineer")

	created, err := st.CreateApplication(t.Context(), userID, dto.CreateApplicationInput{JobID: jobID})
	if err != nil {
		t.Fatalf("CreateApplication err = %v", err)
	}
	_, err = st.CreateApplication(t.Context(), userID, dto.CreateApplicationInput{JobID: jobID})
	if !errors.Is(err, store.ErrApplicationExists) {
		t.Fatalf("duplicate CreateApplication err = %v, want ErrApplicationExists", err)
	}
	status, err := st.CreateApplicationStatus(t.Context(), userID, "Applied", "#6366f1", nil)
	if err != nil {
		t.Fatalf("CreateApplicationStatus err = %v", err)
	}
	for _, in := range []dto.UpdateApplicationInput{
		{StatusID: status.ID}, {StatusID: status.ID, SalaryInfo: "same status, new salary"},
	} {
		if _, err := st.UpdateApplication(t.Context(), userID, created.ID, in); err != nil {
			t.Fatalf("UpdateApplication(%+v) err = %v", in, err)
		}
	}

	rows, err := pool.Query(t.Context(), `SELECT type::text FROM events WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		t.Fatalf("select events err = %v", err)
	}
	types, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatalf("collect events err = %v", err)
	}
	want := []string{events.ApplicationCreated, events.ApplicationStatusChanged}
	if diff := cmp.Diff(want, types); diff != "" {
		t.Errorf("event types (-want +got):\n%s", diff)
	}
}

func TestCreateRecruiterContactMissingJobIsNotFound(t *testing.T) {
	t.Parallel()
	st, _, userID := newStore(t)
	in := dto.LogRecruiterContactInput{
		JobID: "00000000-0000-0000-0000-000000000000", ContactedOn: "2026-10-01",
		Channel: dto.ChannelEmail, Outcome: dto.OutcomeSpoke,
	}
	if _, err := st.CreateRecruiterContact(t.Context(), userID, in); !errors.Is(err, data.ErrNotFound) {
		t.Errorf("CreateRecruiterContact(missing job) err = %v, want ErrNotFound", err)
	}
}

func TestApplyTaskSync(t *testing.T) {
	t.Parallel()

	type env struct {
		st      *store.Store
		pool    *pgxpool.Pool
		userID  string
		jobID   string
		applied string
	}
	setup := func(t *testing.T) env {
		t.Helper()
		st, pool, userID := newStore(t)
		pgtest.InTx(t, pool, true, func(tx pgx.Tx) error {
			if err := st.SeedDefaultStatuses(t.Context(), tx, userID); err != nil {
				return err
			}
			return tasks.NewFacade(pool).SeedDefaults(t.Context(), tx, userID)
		})
		e := env{st: st, pool: pool, userID: userID, jobID: pgtest.InsertJob(t, pool, "Engineer", "Engineer")}
		statuses, err := st.ListApplicationStatusesByUser(t.Context(), userID)
		if err != nil {
			t.Fatalf("ListApplicationStatusesByUser err = %v", err)
		}
		for _, s := range statuses {
			if s.SystemKey == "applied" {
				e.applied = s.ID
			}
		}
		return e
	}
	taskCounts := func(t *testing.T, e env) (open, done int) {
		t.Helper()
		err := e.pool.QueryRow(t.Context(), `SELECT count(*) FILTER (WHERE completed_at IS NULL),
			count(*) FILTER (WHERE completed_at IS NOT NULL) FROM tasks WHERE user_id = $1 AND job_id = $2`,
			e.userID, e.jobID).Scan(&open, &done)
		if err != nil {
			t.Fatalf("count tasks: %v", err)
		}
		return open, done
	}

	t.Run("tracking a job opens one Apply task", func(t *testing.T) {
		t.Parallel()
		e := setup(t)
		if _, err := e.st.CreateApplication(t.Context(), e.userID, dto.CreateApplicationInput{JobID: e.jobID}); err != nil {
			t.Fatalf("CreateApplication err = %v", err)
		}
		if open, done := taskCounts(t, e); open != 1 || done != 0 {
			t.Errorf("open = %d, done = %d, want 1 and 0", open, done)
		}
	})

	t.Run("tracking straight to Applied leaves the task done", func(t *testing.T) {
		t.Parallel()
		e := setup(t)
		if _, err := e.st.CreateApplication(t.Context(), e.userID, dto.CreateApplicationInput{JobID: e.jobID, StatusID: e.applied}); err != nil {
			t.Fatalf("CreateApplication err = %v", err)
		}
		if open, done := taskCounts(t, e); open != 0 || done != 1 {
			t.Errorf("open = %d, done = %d, want 0 and 1", open, done)
		}
	})

	t.Run("setting the application to Applied completes the task", func(t *testing.T) {
		t.Parallel()
		e := setup(t)
		app, err := e.st.CreateApplication(t.Context(), e.userID, dto.CreateApplicationInput{JobID: e.jobID})
		if err != nil {
			t.Fatalf("CreateApplication err = %v", err)
		}
		if _, err := e.st.UpdateApplication(t.Context(), e.userID, app.ID, dto.UpdateApplicationInput{StatusID: e.applied}); err != nil {
			t.Fatalf("UpdateApplication err = %v", err)
		}
		if open, done := taskCounts(t, e); open != 0 || done != 1 {
			t.Errorf("open = %d, done = %d, want 0 and 1", open, done)
		}
	})

	t.Run("moving to another status leaves the task open", func(t *testing.T) {
		t.Parallel()
		e := setup(t)
		app, err := e.st.CreateApplication(t.Context(), e.userID, dto.CreateApplicationInput{JobID: e.jobID})
		if err != nil {
			t.Fatalf("CreateApplication err = %v", err)
		}
		if _, err := e.st.UpdateApplication(t.Context(), e.userID, app.ID, dto.UpdateApplicationInput{SalaryInfo: "50k", StatusID: app.StatusID}); err != nil {
			t.Fatalf("UpdateApplication err = %v", err)
		}
		if open, done := taskCounts(t, e); open != 1 || done != 0 {
			t.Errorf("open = %d, done = %d, want 1 and 0", open, done)
		}
	})
}

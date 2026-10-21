package store_test

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/services/applications"
	"github.com/ollymarsters/job-scraper/internal/services/events"
	"github.com/ollymarsters/job-scraper/internal/services/events/eventstest"
	"github.com/ollymarsters/job-scraper/internal/services/tasks"
	"github.com/ollymarsters/job-scraper/internal/services/tasks/store"
)

type syncEnv struct {
	tasks  *store.Store
	pool   *pgxpool.Pool
	userID string
	jobID  string
}

func newSyncEnv(t *testing.T) syncEnv {
	t.Helper()
	pool := pgtest.New(t)
	apps := applications.New(pool, events.New(pool, eventstest.Unscored()), tasks.NewFacade(pool))
	env := syncEnv{
		tasks:  store.New(pool, apps),
		pool:   pool,
		userID: pgtest.InsertUser(t, pool),
		jobID:  pgtest.InsertJob(t, pool, "Engineer", "Engineer"),
	}
	pgtest.InTx(t, pool, true, func(tx pgx.Tx) error {
		if err := apps.SeedDefaults(t.Context(), tx, env.userID); err != nil {
			return err
		}
		return env.tasks.SeedDefaultTaskTypes(t.Context(), tx, env.userID)
	})
	return env
}

func (e syncEnv) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	if _, err := e.pool.Exec(t.Context(), sql, args...); err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}

func (e syncEnv) trackAs(t *testing.T, systemKey string, appliedAt *string) {
	t.Helper()
	e.exec(t, `INSERT INTO applications (user_id, job_id, status_id, applied_at)
		SELECT $1, $2, s.id, $4::date FROM application_statuses s WHERE s.user_id = $1 AND s.system_key = $3`,
		e.userID, e.jobID, systemKey, appliedAt)
}

func (e syncEnv) addTask(t *testing.T, typeName string) string {
	t.Helper()
	var id string
	err := e.pool.QueryRow(t.Context(), `INSERT INTO tasks (user_id, type_id, job_id)
		SELECT $1, tt.id, $2 FROM task_types tt WHERE tt.user_id = $1 AND tt.name = $3 RETURNING id`,
		e.userID, e.jobID, typeName).Scan(&id)
	if err != nil {
		t.Fatalf("insert %s task: %v", typeName, err)
	}
	return id
}

func (e syncEnv) setDone(t *testing.T, taskID string, done bool) {
	t.Helper()
	if _, err := e.tasks.SetTaskDone(t.Context(), e.userID, taskID, done, time.Now()); err != nil {
		t.Fatalf("SetTaskDone(%v) err = %v", done, err)
	}
}

type applicationRow struct {
	statusKey string
	appliedAt *string
	updatedAt time.Time
}

func (e syncEnv) application(t *testing.T) applicationRow {
	t.Helper()
	var r applicationRow
	err := e.pool.QueryRow(t.Context(), `SELECT s.system_key, a.applied_at::text, a.updated_at
		FROM applications a JOIN application_statuses s ON s.id = a.status_id
		WHERE a.user_id = $1 AND a.job_id = $2`, e.userID, e.jobID).Scan(&r.statusKey, &r.appliedAt, &r.updatedAt)
	if err != nil {
		t.Fatalf("read application: %v", err)
	}
	return r
}

func TestSetTaskDoneSyncsApplication(t *testing.T) {
	t.Parallel()

	t.Run("completing the Apply task sets the application to Applied with a date", func(t *testing.T) {
		t.Parallel()
		env := newSyncEnv(t)
		env.trackAs(t, "draft", nil)

		env.setDone(t, env.addTask(t, "Apply"), true)

		if got := env.application(t); got.statusKey != "applied" || got.appliedAt == nil {
			t.Errorf("application = %+v, want applied with a date", got)
		}
	})

	t.Run("completing the Apply task keeps an existing applied date", func(t *testing.T) {
		t.Parallel()
		env := newSyncEnv(t)
		date := "2026-01-02"
		env.trackAs(t, "draft", &date)

		env.setDone(t, env.addTask(t, "Apply"), true)

		if got := env.application(t); got.appliedAt == nil || *got.appliedAt != date {
			t.Errorf("applied_at = %v, want %s", got.appliedAt, date)
		}
	})

	t.Run("un-completing the task leaves the application Applied", func(t *testing.T) {
		t.Parallel()
		env := newSyncEnv(t)
		env.trackAs(t, "draft", nil)
		id := env.addTask(t, "Apply")

		env.setDone(t, id, true)
		env.setDone(t, id, false)

		if got := env.application(t).statusKey; got != "applied" {
			t.Errorf("application status = %q, want applied", got)
		}
	})

	t.Run("completing the task again does not touch the application", func(t *testing.T) {
		t.Parallel()
		env := newSyncEnv(t)
		env.trackAs(t, "draft", nil)
		id := env.addTask(t, "Apply")
		env.setDone(t, id, true)
		before := env.application(t).updatedAt

		env.setDone(t, id, false)
		env.setDone(t, id, true)

		if after := env.application(t).updatedAt; !after.Equal(before) {
			t.Errorf("application updated_at = %v, want unchanged %v", after, before)
		}
	})

	t.Run("completing a non-Apply task leaves the application alone", func(t *testing.T) {
		t.Parallel()
		env := newSyncEnv(t)
		env.trackAs(t, "draft", nil)

		env.setDone(t, env.addTask(t, "Prep"), true)

		if got := env.application(t).statusKey; got != "draft" {
			t.Errorf("application status = %q, want draft", got)
		}
	})

	t.Run("completing an Apply task for an untracked job succeeds", func(t *testing.T) {
		t.Parallel()
		env := newSyncEnv(t)
		env.setDone(t, env.addTask(t, "Apply"), true)
	})
}

func TestApplicationTracked(t *testing.T) {
	t.Parallel()

	openApplyTasks := func(t *testing.T, env syncEnv) int {
		t.Helper()
		page, err := env.tasks.ListTasks(t.Context(), env.userID, dto.TaskPageOptions{JobID: env.jobID, Limit: 25})
		if err != nil {
			t.Fatalf("ListTasks err = %v", err)
		}
		return page.Total
	}

	t.Run("a rollback discards the task", func(t *testing.T) {
		t.Parallel()
		env := newSyncEnv(t)
		pgtest.InTx(t, env.pool, false, func(tx pgx.Tx) error {
			return tasks.NewFacade(env.pool).ApplicationTracked(t.Context(), tx, env.userID, env.jobID)
		})

		if got := openApplyTasks(t, env); got != 0 {
			t.Errorf("open tasks = %d, want 0", got)
		}
	})

	t.Run("an existing open Apply task is not duplicated", func(t *testing.T) {
		t.Parallel()
		env := newSyncEnv(t)
		env.addTask(t, "Apply")
		pgtest.InTx(t, env.pool, true, func(tx pgx.Tx) error {
			return tasks.NewFacade(env.pool).ApplicationTracked(t.Context(), tx, env.userID, env.jobID)
		})

		if got := openApplyTasks(t, env); got != 1 {
			t.Errorf("open tasks = %d, want 1", got)
		}
	})
}

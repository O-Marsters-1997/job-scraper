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
	status, err := st.CreateApplicationStatus(t.Context(), userID, "Applied", "#6366f1", nil, false)
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

func TestListApplicationsLinksDraft(t *testing.T) {
	t.Parallel()
	st, pool, userID := newStore(t)

	insertDraft := func(jobID, status string, outcome *string, lastError string) string {
		t.Helper()
		var id string
		err := pool.QueryRow(t.Context(), `
			INSERT INTO tailored_cvs (user_id, job_id, base_doc_id, base_tab_id, achievement_ids, status, outcome, last_error)
			VALUES ($1, $2, 'doc', 'tab', '{}', $3, $4, $5) RETURNING id`,
			userID, jobID, status, outcome, lastError).Scan(&id)
		if err != nil {
			t.Fatalf("insert draft: %v", err)
		}
		return id
	}
	kept, discarded := "kept", "discarded"

	keptBeatsLive := pgtest.InsertJob(t, pool, "Kept", "Kept")
	wantKept := insertDraft(keptBeatsLive, "ready", &kept, "")
	insertDraft(keptBeatsLive, "running", nil, "")

	onlyDiscarded := pgtest.InsertJob(t, pool, "Discarded", "Discarded")
	insertDraft(onlyDiscarded, "ready", &discarded, "")

	failedLive := pgtest.InsertJob(t, pool, "Failed", "Failed")
	insertDraft(failedLive, "ready", &discarded, "")
	wantFailed := insertDraft(failedLive, "failed", nil, "boom")

	noDraft := pgtest.InsertJob(t, pool, "None", "None")

	for _, jobID := range []string{keptBeatsLive, onlyDiscarded, failedLive, noDraft} {
		if _, err := st.CreateApplication(t.Context(), userID, dto.CreateApplicationInput{JobID: jobID}); err != nil {
			t.Fatalf("CreateApplication(%s) err = %v", jobID, err)
		}
	}

	rows, err := st.ListApplications(t.Context(), userID, dto.ApplicationsQuery{})
	if err != nil {
		t.Fatalf("ListApplications err = %v", err)
	}
	got := map[string]*dto.LinkedDraft{}
	for _, a := range rows {
		got[a.JobID] = a.Draft
	}
	want := map[string]*dto.LinkedDraft{
		keptBeatsLive: {ID: wantKept, Status: "ready", Outcome: &kept},
		onlyDiscarded: nil,
		failedLive:    {ID: wantFailed, Status: "failed", LastError: "boom"},
		noDraft:       nil,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ListApplications drafts mismatch (-want +got):\n%s", diff)
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

func TestSeededPositiveReplyStatuses(t *testing.T) {
	t.Parallel()
	st, pool, userID := newStore(t)
	pgtest.InTx(t, pool, true, func(tx pgx.Tx) error {
		return st.SeedDefaultStatuses(t.Context(), tx, userID)
	})

	all, err := st.ListApplicationStatusesByUser(t.Context(), userID)
	if err != nil {
		t.Fatalf("ListApplicationStatusesByUser err = %v", err)
	}
	got := map[string]bool{}
	for _, s := range all {
		got[s.Name] = s.CountsAsReply
	}
	want := map[string]bool{"Draft": false, "Applied": false, "Interviewing": true, "Rejected": false, "Offer": true}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("seeded CountsAsReply mismatch (-want +got):\n%s", diff)
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
	status, err := st.CreateApplicationStatus(t.Context(), userID, "Applied", "#6366f1", nil, false)
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

	t.Run("deleting the application removes its open Apply task", func(t *testing.T) {
		t.Parallel()
		e := setup(t)
		app, err := e.st.CreateApplication(t.Context(), e.userID, dto.CreateApplicationInput{JobID: e.jobID})
		if err != nil {
			t.Fatalf("CreateApplication err = %v", err)
		}
		if err := e.st.DeleteApplication(t.Context(), e.userID, app.ID); err != nil {
			t.Fatalf("DeleteApplication err = %v", err)
		}
		if open, done := taskCounts(t, e); open != 0 || done != 0 {
			t.Errorf("open = %d, done = %d, want 0 and 0", open, done)
		}
	})

	t.Run("bulk deleting applications removes their open Apply tasks", func(t *testing.T) {
		t.Parallel()
		e := setup(t)
		app, err := e.st.CreateApplication(t.Context(), e.userID, dto.CreateApplicationInput{JobID: e.jobID})
		if err != nil {
			t.Fatalf("CreateApplication err = %v", err)
		}
		if err := e.st.DeleteApplications(t.Context(), e.userID, []string{app.ID}); err != nil {
			t.Fatalf("DeleteApplications err = %v", err)
		}
		if open, done := taskCounts(t, e); open != 0 || done != 0 {
			t.Errorf("open = %d, done = %d, want 0 and 0", open, done)
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

	t.Run("marking applied sets the chase date from the Applied reply window", func(t *testing.T) {
		t.Parallel()
		e := setup(t)
		if _, err := e.st.UpdateApplicationStatus(t.Context(), e.applied, e.userID, "Applied", "#6366f1", new(5), false); err != nil {
			t.Fatalf("UpdateApplicationStatus err = %v", err)
		}
		app, err := e.st.CreateApplication(t.Context(), e.userID, dto.CreateApplicationInput{JobID: e.jobID})
		if err != nil {
			t.Fatalf("CreateApplication err = %v", err)
		}
		now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
		pgtest.InTx(t, e.pool, true, func(tx pgx.Tx) error {
			return e.st.MarkApplied(t.Context(), tx, e.userID, e.jobID, now)
		})
		apps, err := e.st.ListApplications(t.Context(), e.userID, dto.ApplicationsQuery{})
		if err != nil {
			t.Fatalf("ListApplications err = %v", err)
		}
		want := time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)
		if len(apps) != 1 || apps[0].ID != app.ID || apps[0].ChaseBy == nil || !apps[0].ChaseBy.Equal(want) {
			t.Errorf("ListApplications after MarkApplied = %+v, want one row with ChaseBy %v", apps, want)
		}
	})
}

func TestMoveApplicationAppliedStatus(t *testing.T) {
	t.Parallel()
	st, pool, userID := newStore(t)
	pgtest.InTx(t, pool, true, func(tx pgx.Tx) error {
		if err := st.SeedDefaultStatuses(t.Context(), tx, userID); err != nil {
			return err
		}
		return tasks.NewFacade(pool).SeedDefaults(t.Context(), tx, userID)
	})
	statuses, err := st.ListApplicationStatusesByUser(t.Context(), userID)
	if err != nil {
		t.Fatalf("ListApplicationStatusesByUser err = %v", err)
	}
	var applied, other string
	for _, s := range statuses {
		if s.SystemKey == "applied" {
			applied = s.ID
		} else if s.SystemKey == "" {
			other = s.ID
		}
	}
	jobID := pgtest.InsertJob(t, pool, "Engineer", "Engineer")
	app, err := st.CreateApplication(t.Context(), userID, dto.CreateApplicationInput{JobID: jobID})
	if err != nil {
		t.Fatalf("CreateApplication err = %v", err)
	}
	move := func(in dto.MoveApplicationInput) dto.ApplicationMove {
		t.Helper()
		in.ID = app.ID
		got, err := st.MoveApplication(t.Context(), userID, in)
		if err != nil {
			t.Fatalf("MoveApplication(%+v) err = %v", in, err)
		}
		return got
	}

	got := move(dto.MoveApplicationInput{StatusID: applied})
	if got.Application.AppliedAt == nil || !got.FilledAppliedAt {
		t.Errorf("move into Applied: AppliedAt = %v, FilledAppliedAt = %v, want a date and true", got.Application.AppliedAt, got.FilledAppliedAt)
	}
	var open, done int
	err = pool.QueryRow(t.Context(), `SELECT count(*) FILTER (WHERE completed_at IS NULL),
		count(*) FILTER (WHERE completed_at IS NOT NULL) FROM tasks WHERE user_id = $1 AND job_id = $2`,
		userID, jobID).Scan(&open, &done)
	if err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	if open != 0 || done != 1 {
		t.Errorf("after move into Applied: open = %d, done = %d, want 0 and 1", open, done)
	}

	got = move(dto.MoveApplicationInput{StatusID: other})
	if got.Application.AppliedAt == nil || got.FilledAppliedAt {
		t.Errorf("move out of Applied: AppliedAt = %v, FilledAppliedAt = %v, want the date kept and false", got.Application.AppliedAt, got.FilledAppliedAt)
	}
	got = move(dto.MoveApplicationInput{StatusID: applied, ClearAppliedAt: true})
	if got.Application.AppliedAt != nil || got.FilledAppliedAt {
		t.Errorf("move into Applied with clear: AppliedAt = %v, FilledAppliedAt = %v, want nil and false", got.Application.AppliedAt, got.FilledAppliedAt)
	}
	move(dto.MoveApplicationInput{StatusID: applied})

	var changed int
	err = pool.QueryRow(t.Context(), `SELECT count(*) FROM events WHERE user_id = $1 AND type = $2`, userID, events.ApplicationStatusChanged).Scan(&changed)
	if err != nil {
		t.Fatalf("count events: %v", err)
	}
	if changed != 3 {
		t.Errorf("status change events = %d, want 3 (the repeat move to the current status records none)", changed)
	}
}

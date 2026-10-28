package store_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
)

func exec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(t.Context(), sql, args...); err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}

func queryString(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(t.Context(), sql, args...).Scan(&s); err != nil {
		t.Fatalf("query %q: %v", sql, err)
	}
	return s
}

func trackJob(t *testing.T, pool *pgxpool.Pool, userID, title, chase string) (jobID, appID string) {
	t.Helper()
	jobID = pgtest.InsertJob(t, pool, title, title)
	appID = queryString(t, pool,
		`INSERT INTO applications (user_id, job_id, chase_by) VALUES ($1, $2, NULLIF($3, '')::date) RETURNING id::text`,
		userID, jobID, chase)
	return jobID, appID
}

func taskType(t *testing.T, pool *pgxpool.Pool, userID, name, systemKey string) string {
	t.Helper()
	return queryString(t, pool,
		`INSERT INTO task_types (user_id, name, colour, system_key) VALUES ($1, $2, '#000', NULLIF($3, '')) RETURNING id::text`,
		userID, name, systemKey)
}

func addTask(t *testing.T, pool *pgxpool.Pool, userID, typeID, jobID, notes string) string {
	t.Helper()
	return queryString(t, pool,
		`INSERT INTO tasks (user_id, type_id, job_id, notes) VALUES ($1, $2, $3, $4) RETURNING id::text`,
		userID, typeID, jobID, notes)
}

func addDraft(t *testing.T, pool *pgxpool.Pool, userID, jobID, status, outcome string) string {
	t.Helper()
	return queryString(t, pool,
		`INSERT INTO tailored_cvs (user_id, job_id, base_doc_id, base_tab_id, achievement_ids, status, outcome)
		 VALUES ($1, $2, 'doc1', 'tab1', '{}', $3, NULLIF($4, '')) RETURNING id::text`,
		userID, jobID, status, outcome)
}

func TestStore_ListDue(t *testing.T) {
	t.Parallel()

	t.Run("empty for a user with nothing due", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newStore(t)
		trackJob(t, pool, userID, "Idle", "")

		got, err := st.ListDue(t.Context(), userID)
		if err != nil {
			t.Fatalf("ListDue err = %v", err)
		}
		if len(got.Jobs) != 0 {
			t.Errorf("ListDue jobs = %+v, want none", got.Jobs)
		}
	})

	t.Run("chase-only job carries date, score and band", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newStore(t)
		jobID, appID := trackJob(t, pool, userID, "Chased", "2026-03-04")
		exec(t, pool, `INSERT INTO job_scores (job_id, user_id, suitability_score, band) VALUES ($1, $2, 82, 'great')`, jobID, userID)

		got, err := st.ListDue(t.Context(), userID)
		if err != nil {
			t.Fatalf("ListDue err = %v", err)
		}
		chase := "2026-03-04"
		score := 82
		want := []dto.DueJob{{
			JobID: jobID, Title: "Chased", CompanySlug: got.Jobs[0].CompanySlug, CompanyKind: dto.KindUnclassified, Band: "great", Score: &score,
			ApplicationID: appID, ChaseBy: &chase, Tasks: []dto.DueTask{},
		}}
		if diff := cmp.Diff(want, got.Jobs); diff != "" {
			t.Errorf("ListDue jobs (-want +got):\n%s", diff)
		}
	})

	t.Run("carries the company kind, unclassified when the job has no company", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newStore(t)
		linked, _ := trackJob(t, pool, userID, "Linked", "2026-03-04")
		trackJob(t, pool, userID, "Orphan", "2026-03-05")
		exec(t, pool, `INSERT INTO companies (slug, name, kind) VALUES ('acme-recruit', 'Acme Recruit', 'agency')`)
		exec(t, pool, `UPDATE jobs SET company_id = (SELECT id FROM companies WHERE slug = 'acme-recruit') WHERE id = $1`, linked)

		got, err := st.ListDue(t.Context(), userID)
		if err != nil {
			t.Fatalf("ListDue err = %v", err)
		}
		var kinds []dto.CompanyKind
		for _, j := range got.Jobs {
			kinds = append(kinds, j.CompanyKind)
		}
		if diff := cmp.Diff([]dto.CompanyKind{dto.KindAgency, dto.KindUnclassified}, kinds); diff != "" {
			t.Errorf("ListDue kinds (-want +got):\n%s", diff)
		}
	})

	t.Run("task-only job lists open non-apply tasks and exposes the apply task", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newStore(t)
		jobID, _ := trackJob(t, pool, userID, "Tasked", "")
		apply := taskType(t, pool, userID, "Apply", "apply")
		reach := taskType(t, pool, userID, "Reach out", "")
		applyID := addTask(t, pool, userID, apply, jobID, "")
		reachID := addTask(t, pool, userID, reach, jobID, "email Sam")
		doneID := addTask(t, pool, userID, reach, jobID, "")
		exec(t, pool, `UPDATE tasks SET completed_at = NOW() WHERE id = $1`, doneID)
		finishID := addTask(t, pool, userID, taskType(t, pool, userID, "Finish application", "finish"), jobID, "needs a cover letter")

		got, err := st.ListDue(t.Context(), userID)
		if err != nil {
			t.Fatalf("ListDue err = %v", err)
		}
		if len(got.Jobs) != 1 {
			t.Fatalf("ListDue jobs = %d, want 1", len(got.Jobs))
		}
		j := got.Jobs[0]
		if j.ApplyTaskID == nil || *j.ApplyTaskID != applyID {
			t.Errorf("ApplyTaskID = %v, want %s", j.ApplyTaskID, applyID)
		}
		wantTasks := []dto.DueTask{
			{ID: reachID, TypeName: "Reach out", Notes: "email Sam"},
			{ID: finishID, TypeName: "Finish application", SystemKey: dto.TaskTypeFinish, Notes: "needs a cover letter"},
		}
		if diff := cmp.Diff(wantTasks, j.Tasks); diff != "" {
			t.Errorf("Tasks (-want +got):\n%s", diff)
		}
		if j.Score != nil || j.Band != "" {
			t.Errorf("unscored job Score/Band = %v/%q, want nil/empty", j.Score, j.Band)
		}
	})

	t.Run("draft maps each status while the apply task is open", func(t *testing.T) {
		t.Parallel()
		cases := map[string]dto.DraftState{
			"pending": dto.DraftStatePending, "running": dto.DraftStatePending,
			"ready": dto.DraftStateReady, "keeping": dto.DraftStateReady,
			"failed": dto.DraftStateFailed,
		}
		for status, wantState := range cases {
			t.Run(status, func(t *testing.T) {
				t.Parallel()
				st, pool, userID := newStore(t)
				jobID, _ := trackJob(t, pool, userID, "Drafted", "")
				addTask(t, pool, userID, taskType(t, pool, userID, "Apply", "apply"), jobID, "")
				draftID := addDraft(t, pool, userID, jobID, status, "")
				docID := queryString(t, pool, `INSERT INTO tracked_docs (user_id, doc_id) VALUES ($1, 'doc1') RETURNING id::text`, userID)
				exec(t, pool, `INSERT INTO tracked_doc_tabs (tracked_doc_id, tab_id, title) VALUES ($1, 'tab1', 'Backend')`, docID)

				got, err := st.ListDue(t.Context(), userID)
				if err != nil {
					t.Fatalf("ListDue err = %v", err)
				}
				if len(got.Jobs) != 1 {
					t.Fatalf("ListDue jobs = %d, want 1", len(got.Jobs))
				}
				want := &dto.DueDraft{ID: draftID, State: wantState, CVTitle: "Backend"}
				if diff := cmp.Diff(want, got.Jobs[0].Draft); diff != "" {
					t.Errorf("Draft (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("ticking the apply task drops the draft", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newStore(t)
		jobID, _ := trackJob(t, pool, userID, "Applied", "2026-03-04")
		applyID := addTask(t, pool, userID, taskType(t, pool, userID, "Apply", "apply"), jobID, "")
		addDraft(t, pool, userID, jobID, "ready", "")
		exec(t, pool, `UPDATE tasks SET completed_at = NOW() WHERE id = $1`, applyID)

		got, err := st.ListDue(t.Context(), userID)
		if err != nil {
			t.Fatalf("ListDue err = %v", err)
		}
		if len(got.Jobs) != 1 || got.Jobs[0].Draft != nil || got.Jobs[0].ApplyTaskID != nil {
			t.Errorf("ListDue = %+v, want one job with no draft and no apply task", got.Jobs)
		}
	})

	t.Run("a kept draft surfaces as the kept draft id, not the draft", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newStore(t)
		jobID, _ := trackJob(t, pool, userID, "Decided", "")
		addTask(t, pool, userID, taskType(t, pool, userID, "Apply", "apply"), jobID, "")
		keptID := addDraft(t, pool, userID, jobID, "ready", "kept")
		addDraft(t, pool, userID, jobID, "ready", "discarded")

		got, err := st.ListDue(t.Context(), userID)
		if err != nil {
			t.Fatalf("ListDue err = %v", err)
		}
		if len(got.Jobs) != 1 {
			t.Fatalf("ListDue jobs = %d, want 1", len(got.Jobs))
		}
		j := got.Jobs[0]
		if j.Draft != nil || j.KeptDraftID == nil || *j.KeptDraftID != keptID {
			t.Errorf("Draft/KeptDraftID = %+v/%v, want nil/%s", j.Draft, j.KeptDraftID, keptID)
		}
	})

	t.Run("ticking the apply task drops the kept draft id", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newStore(t)
		jobID, _ := trackJob(t, pool, userID, "Applied", "2026-03-04")
		applyID := addTask(t, pool, userID, taskType(t, pool, userID, "Apply", "apply"), jobID, "")
		addDraft(t, pool, userID, jobID, "ready", "kept")
		exec(t, pool, `UPDATE tasks SET completed_at = NOW() WHERE id = $1`, applyID)

		got, err := st.ListDue(t.Context(), userID)
		if err != nil {
			t.Fatalf("ListDue err = %v", err)
		}
		if len(got.Jobs) != 1 || got.Jobs[0].KeptDraftID != nil {
			t.Errorf("ListDue = %+v, want one job with no kept draft id", got.Jobs)
		}
	})

	t.Run("another user's data is excluded", func(t *testing.T) {
		t.Parallel()
		st, pool, userID := newStore(t)
		otherID := pgtest.InsertUser(t, pool)
		jobID, _ := trackJob(t, pool, otherID, "Theirs", "2026-03-04")
		addTask(t, pool, otherID, taskType(t, pool, otherID, "Apply", "apply"), jobID, "")
		addDraft(t, pool, otherID, jobID, "ready", "")

		got, err := st.ListDue(t.Context(), userID)
		if err != nil {
			t.Fatalf("ListDue err = %v", err)
		}
		if len(got.Jobs) != 0 {
			t.Errorf("ListDue jobs = %+v, want none", got.Jobs)
		}
	})
}

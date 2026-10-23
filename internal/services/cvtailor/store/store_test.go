package store_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store"
)

const missingID = "00000000-0000-0000-0000-00000000dead"

func TestStoreContract(t *testing.T) {
	t.Parallel()
	cvtailortest.RunStoreContract(t, func(t *testing.T) cvtailortest.Fixture {
		t.Helper()
		pool := pgtest.New(t)
		return cvtailortest.Fixture{
			Store: store.New(pool), UserID: pgtest.InsertUser(t, pool), Other: pgtest.InsertUser(t, pool), JobID: pgtest.InsertJob(t, pool, "Role", "fp-1"),
			TrackLetterTab: func(t *testing.T, userID, docID, tabID string) {
				t.Helper()
				_, err := pool.Exec(t.Context(), `WITH d AS (
					INSERT INTO tracked_docs (user_id, doc_id, kind) VALUES ($1, $2, 'letter') RETURNING id
				) INSERT INTO tracked_doc_tabs (tracked_doc_id, tab_id) SELECT id, $3 FROM d`, userID, docID, tabID)
				if err != nil {
					t.Fatal(err)
				}
			},
		}
	})
}

func TestJobBrief(t *testing.T) {
	t.Parallel()
	st, pool, _ := newStore(t)
	jobID := pgtest.InsertJob(t, pool, "Backend Engineer", "fp-1")

	got, err := st.JobBrief(t.Context(), jobID)

	want := dto.JobBrief{Title: "Backend Engineer", Company: "acme", Description: "Build things in Go"}
	if diff := cmp.Diff(want, got); err != nil || diff != "" {
		t.Errorf("JobBrief() = %+v, %v (-want +got):\n%s", got, err, diff)
	}
}

func newStore(t *testing.T) (*store.Store, *pgxpool.Pool, string) {
	t.Helper()
	pool := pgtest.New(t)
	return store.New(pool), pool, pgtest.InsertUser(t, pool)
}

func TestClaimDraftReclaimsAfterLeaseExpiry(t *testing.T) {
	t.Parallel()
	st, pool, uid := newStore(t)
	ctx := t.Context()
	jobID := pgtest.InsertJob(t, pool, "Role", "fp-1")
	d, err := st.CreateDraft(ctx, uid, dto.DraftInput{JobID: jobID, DocID: "doc", TabID: "t.0"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := st.ClaimDraft(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if first.JobDescription != "Build things in Go" || first.JobFingerprint != "fp-1" {
		t.Errorf("ClaimDraft() job facts = %q, %q, want the Job's description and fingerprint", first.JobDescription, first.JobFingerprint)
	}
	if _, err := st.ClaimDraft(ctx); !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("ClaimDraft() under a live lease err = %v, want ErrNotFound", err)
	}

	if _, err := pool.Exec(ctx, `UPDATE tailored_cvs SET lease_until = NOW() - interval '1 second' WHERE id = $1`, d.ID); err != nil {
		t.Fatal(err)
	}
	second, err := st.ClaimDraft(ctx)
	if err != nil {
		t.Fatalf("ClaimDraft() after lease expiry err = %v, want the crashed Draft", err)
	}
	if second.ID != d.ID || second.Attempts != 2 {
		t.Errorf("ClaimDraft() = %+v, want Draft %s on attempt 2", second, d.ID)
	}
	if err := st.CompleteDraft(ctx, first, dto.DraftResult{EditSet: json.RawMessage(`{}`)}); !errors.Is(err, store.ErrDraftNotFound) {
		t.Errorf("CompleteDraft() by the crashed claim err = %v, want ErrDraftNotFound", err)
	}
}

func TestCreateDraftUnknownJobIsNotFound(t *testing.T) {
	t.Parallel()
	st, _, uid := newStore(t)

	_, err := st.CreateDraft(t.Context(), uid, dto.DraftInput{JobID: missingID, DocID: "doc", TabID: "t.0"})

	if !errors.Is(err, store.ErrJobNotFound) {
		t.Errorf("CreateDraft() err = %v, want ErrJobNotFound", err)
	}
}

func TestCompleteKeepRecordsAchievementOutcomes(t *testing.T) {
	t.Parallel()
	st, pool, uid := newStore(t)
	ctx := t.Context()
	jobID := pgtest.InsertJob(t, pool, "Role", "fp-1")
	pos, err := st.CreatePosition(ctx, uid, dto.PositionInput{Employer: "Acme", Title: "Engineer"})
	if err != nil {
		t.Fatal(err)
	}
	kept, err := st.CreateAchievement(ctx, uid, dto.AchievementInput{PositionID: pos.ID, Text: "Cut latency"})
	if err != nil {
		t.Fatal(err)
	}
	unseen, err := st.CreateAchievement(ctx, uid, dto.AchievementInput{PositionID: pos.ID, Text: "Mentored"})
	if err != nil {
		t.Fatal(err)
	}
	in := dto.DraftInput{JobID: jobID, DocID: "doc", TabID: "t.0"}
	yes, no := 0.7, 0.1
	outcomes := []dto.AchievementOutcome{
		{AchievementID: kept.ID, PYes: &yes, PNo: &no, OnSourceCV: true, InFinalCV: true, Reworded: true},
		{AchievementID: unseen.ID, OutscoredKeptButLeftOut: true},
	}
	d, err := st.CreateDraft(ctx, uid, in)
	if err != nil {
		t.Fatalf("CreateDraft() err = %v", err)
	}
	if err := st.CompleteDraft(ctx, mustClaim(t, st), dto.DraftResult{EditSet: json.RawMessage(`{}`), DraftDocID: "copy"}); err != nil {
		t.Fatal(err)
	}
	countOutcomes := func() (n int) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM achievement_outcomes WHERE draft_id = $1`, d.ID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	if _, err := st.QueueKeep(ctx, uid, d.ID); err != nil {
		t.Fatal(err)
	}
	claim := mustClaim(t, st)
	if got := countOutcomes(); got != 0 {
		t.Fatalf("outcomes before CompleteKeep = %d, want 0", got)
	}
	stale := claim
	stale.Attempts++
	if err := st.CompleteKeep(ctx, stale, outcomes); !errors.Is(err, store.ErrDraftNotFound) {
		t.Fatalf("CompleteKeep(stale claim) err = %v, want ErrDraftNotFound", err)
	}
	if got := countOutcomes(); got != 0 {
		t.Fatalf("outcomes after a stale CompleteKeep = %d, want 0", got)
	}
	if err := st.CompleteKeep(ctx, claim, outcomes); err != nil {
		t.Fatalf("CompleteKeep() err = %v", err)
	}

	type row struct {
		AchievementID                                  string
		PYes, PNo                                      *float64
		OnSourceCV, InFinalCV, OutscoredKept, Reworded bool
	}
	rows, err := pool.Query(ctx, `SELECT achievement_id::text, p_yes, p_no, on_source_cv, in_final_cv, outscored_kept_but_left_out, reworded
		FROM achievement_outcomes WHERE draft_id = $1 AND user_id = $2 AND job_id = $3 ORDER BY in_final_cv DESC`, d.ID, uid, jobID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.AchievementID, &r.PYes, &r.PNo, &r.OnSourceCV, &r.InFinalCV, &r.OutscoredKept, &r.Reworded); err != nil {
			t.Fatal(err)
		}
		got = append(got, r)
	}
	want := []row{
		{AchievementID: kept.ID, PYes: &yes, PNo: &no, OnSourceCV: true, InFinalCV: true, Reworded: true},
		{AchievementID: unseen.ID, OutscoredKept: true},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("achievement_outcomes mismatch (-want +got):\n%s", diff)
	}
}

func TestListPositionsAchievementStats(t *testing.T) {
	t.Parallel()
	st, pool, uid := newStore(t)
	ctx := t.Context()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	status := func(name string, key *string, reply bool) (id string) {
		t.Helper()
		err := pool.QueryRow(ctx, `INSERT INTO application_statuses (user_id, name, colour, counts_as_reply, system_key)
			VALUES ($1, $2, '#000', $3, $4) RETURNING id::text`, uid, name, reply, key).Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	applied := "applied"
	appliedID, interviewID, rejectedID := status("Applied", &applied, false), status("Interview", nil, true), status("Rejected", nil, false)
	pos, err := st.CreatePosition(ctx, uid, dto.PositionInput{Employer: "Acme", Title: "Engineer"})
	if err != nil {
		t.Fatal(err)
	}
	ach, err := st.CreateAchievement(ctx, uid, dto.AchievementInput{PositionID: pos.ID, Text: "Cut latency"})
	if err != nil {
		t.Fatal(err)
	}
	unused, err := st.CreateAchievement(ctx, uid, dto.AchievementInput{PositionID: pos.ID, Text: "Mentored"})
	if err != nil {
		t.Fatal(err)
	}
	cv := func(jobID, outcome string, inFinal, leftOut bool) {
		t.Helper()
		var draftID string
		err := pool.QueryRow(ctx, `INSERT INTO tailored_cvs (user_id, job_id, base_doc_id, base_tab_id, achievement_ids, status, outcome)
			VALUES ($1, $2, 'd', 't', '{}', 'ready', $3) RETURNING id::text`, uid, jobID, outcome).Scan(&draftID)
		if err != nil {
			t.Fatal(err)
		}
		exec(`INSERT INTO achievement_outcomes (draft_id, achievement_id, user_id, job_id, on_source_cv, in_final_cv, outscored_kept_but_left_out, reworded)
			VALUES ($1, $2, $3, $4, FALSE, $5, $6, FALSE)`, draftID, ach.ID, uid, jobID, inFinal, leftOut)
	}
	application := func(jobID, statusID string) (id string) {
		t.Helper()
		err := pool.QueryRow(ctx, `INSERT INTO applications (user_id, job_id, status_id) VALUES ($1, $2, $3) RETURNING id::text`, uid, jobID, statusID).Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	changedTo := func(appID, statusID string) {
		t.Helper()
		exec(`INSERT INTO events (user_id, type, subject_id, props) VALUES ($1, 'application_status_changed', $2, jsonb_build_object('to_status_id', $3::text))`, uid, appID, statusID)
	}

	movedOn := pgtest.InsertJob(t, pool, "Replied then rejected", "fp-1")
	cv(movedOn, "kept", true, false)
	app := application(movedOn, rejectedID)
	changedTo(app, appliedID)
	changedTo(app, interviewID)
	changedTo(app, rejectedID)
	createdApplied := pgtest.InsertJob(t, pool, "Created applied", "fp-2")
	cv(createdApplied, "kept", true, true)
	application(createdApplied, appliedID)
	notApplied := pgtest.InsertJob(t, pool, "Never applied", "fp-3")
	cv(notApplied, "kept", false, true)
	application(notApplied, interviewID)
	discarded := pgtest.InsertJob(t, pool, "Discarded draft", "fp-4")
	cv(discarded, "discarded", true, false)

	got, err := st.ListPositions(ctx, uid)
	if err != nil {
		t.Fatal(err)
	}

	want := []*dto.AchievementStats{{InCVs: 2, LeftOut: 2, Applied: 2, Replied: 1}, nil}
	var stats []*dto.AchievementStats
	for _, a := range got[0].Achievements {
		stats = append(stats, a.Stats)
	}
	if diff := cmp.Diff(want, stats); diff != "" {
		t.Errorf("ListPositions() stats for %s, %s (-want +got):\n%s", ach.ID, unused.ID, diff)
	}
}

func mustClaim(t *testing.T, st *store.Store) dto.DraftClaim {
	t.Helper()
	c, err := st.ClaimDraft(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestOnlyOneKeptTailoredCVPerJob(t *testing.T) {
	t.Parallel()
	_, pool, uid := newStore(t)
	ctx := t.Context()
	jobID := pgtest.InsertJob(t, pool, "Role", "fp-1")
	insert := func(outcome string) error {
		_, err := pool.Exec(ctx, `INSERT INTO tailored_cvs (user_id, job_id, base_doc_id, base_tab_id, achievement_ids, outcome)
			VALUES ($1, $2, 'doc', 't.0', '{}', $3)`, uid, jobID, outcome)
		return err
	}
	if err := insert("kept"); err != nil {
		t.Fatal(err)
	}
	if err := insert("discarded"); err != nil {
		t.Errorf("second discarded row err = %v, want it allowed", err)
	}
	if err := insert("kept"); !data.IsUniqueViolation(err) {
		t.Errorf("second kept row err = %v, want a unique violation", err)
	}
}

func TestImportPositionsIsAllOrNothing(t *testing.T) {
	t.Parallel()
	st, pool, uid := newStore(t)
	ctx := t.Context()

	bad := "not-a-date"
	_, err := st.ImportPositions(ctx, uid, []dto.ImportPosition{
		{Employer: "Broken", Title: "Engineer", StartDate: &bad},
		{Employer: "Acme", Title: "Engineer", Achievements: []string{"shipped"}},
	})
	if err == nil {
		t.Fatal("ImportPositions() error = nil, want an error for the bad date")
	}

	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM positions`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("positions rows after failed import = %d, want 0", n)
	}
}

func TestJobDescription(t *testing.T) {
	t.Parallel()
	st, pool, _ := newStore(t)
	jobID := pgtest.InsertJob(t, pool, "Role", "fp-1")

	got, err := st.JobDescription(t.Context(), jobID)
	if err != nil || got != "Build things in Go" {
		t.Errorf("JobDescription() = %q, %v, want the Job's description", got, err)
	}
	if _, err := st.JobDescription(t.Context(), missingID); !errors.Is(err, store.ErrJobNotFound) {
		t.Errorf("JobDescription(unknown) err = %v, want ErrJobNotFound", err)
	}
}

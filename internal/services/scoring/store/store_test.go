package store_test

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/scoringtest"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/store"
)

var seedCounter atomic.Int64

func newStore(t *testing.T) (*store.Store, *pgxpool.Pool) {
	t.Helper()
	pool := pgtest.New(t)
	return store.New(pool), pool
}

func countRows(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(t.Context(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("count rows (%s): %v", sql, err)
	}
	return n
}

func exec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(t.Context(), sql, args...); err != nil {
		t.Fatalf("exec (%s): %v", sql, err)
	}
}

func insertScore(t *testing.T, pool *pgxpool.Pool, jobID, userID string) {
	t.Helper()
	exec(t, pool, `INSERT INTO job_scores (job_id, user_id, suitability_score, breakdown) VALUES ($1, $2, 50, '[]')`, jobID, userID)
}

func insertAnswer(t *testing.T, pool *pgxpool.Pool, jobID, fingerprint, hash, model string) {
	t.Helper()
	exec(t, pool,
		`INSERT INTO option_answers (job_id, fingerprint, question_hash, model, p_yes, p_no, p_not_stated, confidence)
		 VALUES ($1, $2, $3, $4, 0.9, 0.05, 0.05, 0.9)`, jobID, fingerprint, hash, model)
}

func opsState(t *testing.T, st *store.Store) dto.OpsState {
	t.Helper()
	state, err := st.OpsState(t.Context())
	if err != nil {
		t.Fatalf("OpsState() error = %v", err)
	}
	return state
}

func closeJob(t *testing.T, pool *pgxpool.Pool, jobID string) {
	t.Helper()
	exec(t, pool, "UPDATE jobs SET closed_at = NOW() WHERE id = $1", jobID)
}

func insertEffect(t *testing.T, pool *pgxpool.Pool, jobID, fingerprint string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(t.Context(),
		`INSERT INTO effect_outbox (job_id, fingerprint) VALUES ($1, $2) RETURNING id`,
		jobID, fingerprint).Scan(&id)
	if err != nil {
		t.Fatalf("insert effect: %v", err)
	}
	return id
}

func TestScoringStoreContract(t *testing.T) {
	scoringtest.RunStoreContract(t, func(t *testing.T) scoringtest.Fixture {
		t.Helper()
		st, pool := newStore(t)
		return scoringtest.Fixture{Store: st, NewUser: func() string { return pgtest.InsertUser(t, pool) }}
	})
}

func TestSearchConfig_UpsertThenGetRoundTrips(t *testing.T) {
	st, pool := newStore(t)
	ctx := t.Context()
	userID := pgtest.InsertUser(t, pool)

	saved, err := st.UpsertSearchConfig(ctx, dto.SearchConfig{
		UserID: userID, NotifyThreshold: 70,
		Preferences: dto.Preferences{Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "manual"}}},
	})
	if err != nil {
		t.Fatalf("UpsertSearchConfig() err = %v", err)
	}
	got, err := st.GetSearchConfig(ctx, userID)
	if err != nil {
		t.Fatalf("GetSearchConfig() err = %v", err)
	}
	if diff := cmp.Diff(saved, got); diff != "" {
		t.Errorf("GetSearchConfig() differs from the upserted config (-saved +got):\n%s", diff)
	}
	if got.UserID != userID || got.NotifyThreshold != 70 || len(got.Preferences.Picks) != 1 {
		t.Errorf("GetSearchConfig() = %+v, want user %q, threshold 70, one pick", got, userID)
	}
}

func TestAddScoringOption_QueuesBackfillForScoredOpenJobsOnly(t *testing.T) {
	st, pool := newStore(t)
	userID := pgtest.InsertUser(t, pool)
	scored := pgtest.InsertJob(t, pool, "Engineer", "fp-scored")
	closed := pgtest.InsertJob(t, pool, "Engineer", "fp-closed")
	unscored := pgtest.InsertJob(t, pool, "Engineer", "fp-unscored")
	insertScore(t, pool, scored, userID)
	insertScore(t, pool, closed, userID)
	closeJob(t, pool, closed)

	if err := st.AddScoringOption(t.Context(), "tech:zig", "tech", "Zig", "Does the role use Zig?"); err != nil {
		t.Fatalf("AddScoringOption() err = %v", err)
	}

	for jobID, want := range map[string]int{scored: 1, closed: 0, unscored: 0} {
		if got := countRows(t, pool, "SELECT count(*) FROM effect_outbox WHERE job_id = $1", jobID); got != want {
			t.Errorf("effects for job %s = %d, want %d", jobID, got, want)
		}
	}
}

func TestListInterestedConfigs_JoinsTrackedCompany(t *testing.T) {
	st, pool := newStore(t)
	jobID := pgtest.InsertJob(t, pool, "Engineer", "fp-1")
	_, userID := insertTrackedCompany(t, pool, "acme")

	configs, err := st.ListInterestedConfigs(t.Context(), jobID)
	if err != nil {
		t.Fatalf("ListInterestedConfigs() err = %v", err)
	}
	if len(configs) != 1 || configs[0].UserID != userID {
		t.Fatalf("ListInterestedConfigs() = %+v, want one config for %q", configs, userID)
	}
}

func TestListInterestedConfigs_FlagsNewCompany(t *testing.T) {
	st, pool := newStore(t)
	jobID := pgtest.InsertJob(t, pool, "Engineer", "fp-1")
	companyID, _ := insertTrackedCompany(t, pool, "acme")

	for state, want := range map[string]bool{"kept": false, "new": true} {
		exec(t, pool, `UPDATE tracked_companies SET review_state = $1 WHERE company_id = $2`, state, companyID)
		configs, err := st.ListInterestedConfigs(t.Context(), jobID)
		if err != nil || len(configs) != 1 {
			t.Fatalf("ListInterestedConfigs() = %+v, %v, want one config", configs, err)
		}
		if configs[0].CompanyIsNew != want {
			t.Errorf("CompanyIsNew with review_state %q = %v, want %v", state, configs[0].CompanyIsNew, want)
		}
	}
}

func TestClaimAnswerEffect_ThenCompleteWritesAnswersAndScores(t *testing.T) {
	st, pool := newStore(t)
	ctx := t.Context()
	userID := pgtest.InsertUser(t, pool)
	jobID := pgtest.InsertJob(t, pool, "Engineer", "fp-1")
	insertEffect(t, pool, jobID, "fp-1")

	effect, err := st.ClaimAnswerEffect(t.Context())
	if err != nil {
		t.Fatalf("ClaimAnswerEffect() err = %v", err)
	}
	if effect.JobID != jobID || effect.Attempts != 1 {
		t.Fatalf("ClaimAnswerEffect() = %+v, want jobID %q with attempts 1", effect, jobID)
	}

	saved, err := st.CompleteAnswerEffect(ctx, effect,
		map[string]dto.Answer{"hash-1": {PYes: 0.9, PNo: 0.05, PNotStated: 0.05}},
		[]dto.JobScore{{JobID: jobID, UserID: userID, Score: 80, Rows: []dto.ScoreRow{}}},
	)
	if err != nil {
		t.Fatalf("CompleteAnswerEffect() err = %v", err)
	}
	if diff := cmp.Diff([]string{userID}, saved); diff != "" {
		t.Errorf("CompleteAnswerEffect() saved users (-want +got):\n%s", diff)
	}

	answers, err := st.ListAnswers(ctx, jobID, "fp-1", "typesafe/jev-1.13")
	if err != nil {
		t.Fatalf("ListAnswers() err = %v", err)
	}
	if len(answers) != 1 || answers["hash-1"].PYes != float64(float32(0.9)) {
		t.Errorf("ListAnswers() = %+v, want one cached answer", answers)
	}

	status, err := st.GetScoringStatus(ctx, userID)
	if err != nil {
		t.Fatalf("GetScoringStatus() err = %v", err)
	}
	if status.Pending != 0 {
		t.Errorf("GetScoringStatus().Pending = %d, want 0 after completion", status.Pending)
	}
}

func TestCompleteAnswerEffect_CommitsBothUsersScoresTogether(t *testing.T) {
	st, pool := newStore(t)
	alice := pgtest.InsertUser(t, pool)
	bob := pgtest.InsertUser(t, pool)
	jobID := pgtest.InsertJob(t, pool, "Engineer", "fp-1")
	insertEffect(t, pool, jobID, "fp-1")

	effect, err := st.ClaimAnswerEffect(t.Context())
	if err != nil {
		t.Fatalf("ClaimAnswerEffect() err = %v", err)
	}

	scores := []dto.JobScore{
		{JobID: jobID, UserID: alice, Score: 79, Rows: []dto.ScoreRow{{Key: "tech:go", Stance: "nice", Resolved: "yes", Effect: "meets"}}},
		{JobID: jobID, UserID: bob, Score: 21, Rows: []dto.ScoreRow{{Key: "tech:go", Stance: "avoid", Resolved: "yes", Effect: "misses"}}},
	}
	saved, err := st.CompleteAnswerEffect(t.Context(), effect, map[string]dto.Answer{"hash-go": {PYes: 0.9, PNo: 0.05, PNotStated: 0.05}}, scores)
	if err != nil {
		t.Fatalf("CompleteAnswerEffect() err = %v", err)
	}
	if len(saved) != 2 {
		t.Errorf("CompleteAnswerEffect() saved users = %v, want 2", saved)
	}
	if got := countRows(t, pool, "SELECT count(*) FROM job_scores WHERE job_id = $1", jobID); got != 2 {
		t.Errorf("job_scores rows = %d, want 2", got)
	}
}

func TestCompleteAnswerEffect_FingerprintMismatchWritesNothing(t *testing.T) {
	st, pool := newStore(t)
	userID := pgtest.InsertUser(t, pool)
	jobID := pgtest.InsertJob(t, pool, "Engineer", "fp-1")
	insertEffect(t, pool, jobID, "fp-1")

	effect, err := st.ClaimAnswerEffect(t.Context())
	if err != nil {
		t.Fatalf("ClaimAnswerEffect() err = %v", err)
	}
	exec(t, pool, "UPDATE jobs SET content_fingerprint = 'fp-2' WHERE id = $1", jobID)

	saved, err := st.CompleteAnswerEffect(t.Context(), effect,
		map[string]dto.Answer{"hash-go": {PYes: 0.9, PNo: 0.05, PNotStated: 0.05}},
		[]dto.JobScore{{JobID: jobID, UserID: userID, Score: 79}},
	)
	if err != nil {
		t.Fatalf("CompleteAnswerEffect() err = %v", err)
	}
	if len(saved) != 0 {
		t.Errorf("CompleteAnswerEffect() saved users = %v, want none (fingerprint moved on)", saved)
	}
	if got := countRows(t, pool, "SELECT count(*) FROM option_answers WHERE job_id = $1", jobID); got != 0 {
		t.Errorf("option_answers rows = %d, want 0", got)
	}
	if got := countRows(t, pool, "SELECT count(*) FROM job_scores WHERE job_id = $1", jobID); got != 0 {
		t.Errorf("job_scores rows = %d, want 0", got)
	}
	if got := countRows(t, pool, "SELECT count(*) FROM effect_outbox WHERE id = $1 AND status = 'done'", effect.ID); got != 1 {
		t.Errorf("done effects = %d, want 1 (a stale effect is still marked done)", got)
	}
}

func TestFailAnswerEffect_ReschedulesForRetry(t *testing.T) {
	st, pool := newStore(t)
	jobID := pgtest.InsertJob(t, pool, "Engineer", "fp-1")
	insertEffect(t, pool, jobID, "fp-1")

	effect, err := st.ClaimAnswerEffect(t.Context())
	if err != nil {
		t.Fatalf("ClaimAnswerEffect() err = %v", err)
	}
	if err := st.FailAnswerEffect(t.Context(), effect.ID, effect.Attempts, dto.ScoringFailure{Reason: "boom"}); err != nil {
		t.Fatalf("FailAnswerEffect() err = %v", err)
	}

	got := countRows(t, pool, `SELECT count(*) FROM effect_outbox WHERE id = $1 AND status = 'pending' AND last_error = 'boom'`, effect.ID)
	if got != 1 {
		t.Errorf("pending effects with last_error boom = %d, want 1", got)
	}
}

func TestListScoringInputs(t *testing.T) {
	const model = "typesafe/jev-1.13"
	tests := []struct {
		name        string
		answerModel string
		wantAnswers int
	}{
		{"includes answers under the current model", model, 1},
		{"ignores answers under another model", "old-model", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, pool := newStore(t)
			userID := pgtest.InsertUser(t, pool)
			jobID := pgtest.InsertJob(t, pool, "Engineer", "fp-1")
			insertScore(t, pool, jobID, userID)
			insertAnswer(t, pool, jobID, "fp-1", "hash-1", tt.answerModel)

			inputs, err := st.ListScoringInputs(t.Context(), userID, model)
			if err != nil {
				t.Fatalf("ListScoringInputs() err = %v", err)
			}
			if len(inputs) != 1 || inputs[0].Job.ID != jobID || len(inputs[0].Answers) != tt.wantAnswers {
				t.Errorf("ListScoringInputs() = %+v, want one job with %d answers", inputs, tt.wantAnswers)
			}
		})
	}
}

func TestListCompanyAnswers(t *testing.T) {
	const model = "typesafe/jev-1.13"
	st, pool := newStore(t)
	companyID, _ := insertTrackedCompany(t, pool, "acme")
	insertCompanyJob := func(url, fingerprint, closedAt string) string {
		var id string
		err := pool.QueryRow(t.Context(),
			`INSERT INTO jobs (title, location, url, company_slug, company_id, source, updated_at, content_fingerprint, closed_at)
			 VALUES ('Engineer', 'Remote', $1, 'acme', $2, 'greenhouse', NOW(), $3, NULLIF($4, '')::timestamptz) RETURNING id`,
			url, companyID, fingerprint, closedAt).Scan(&id)
		if err != nil {
			t.Fatalf("insert job: %v", err)
		}
		return id
	}
	answered := insertCompanyJob("https://example.com/a", "fp-1", "")
	insertCompanyJob("https://example.com/b", "fp-1", "")
	stale := insertCompanyJob("https://example.com/c", "fp-2", "")
	closed := insertCompanyJob("https://example.com/d", "fp-1", "2026-01-01T00:00:00Z")
	insertAnswer(t, pool, answered, "fp-1", "hash-1", model)
	insertAnswer(t, pool, answered, "fp-1", "hash-2", "old-model")
	insertAnswer(t, pool, stale, "fp-1", "hash-1", model)
	insertAnswer(t, pool, closed, "fp-1", "hash-1", model)

	got, err := st.ListCompanyAnswers(t.Context(), []string{companyID}, model)
	if err != nil {
		t.Fatalf("ListCompanyAnswers() err = %v", err)
	}
	jobs := got[companyID]
	if len(jobs) != 3 {
		t.Fatalf("ListCompanyAnswers() = %d open jobs, want 3", len(jobs))
	}
	withAnswers := 0
	for _, answers := range jobs {
		if _, ok := answers["hash-1"]; ok {
			withAnswers++
		}
		if _, ok := answers["hash-2"]; ok {
			t.Errorf("ListCompanyAnswers() included an answer under another model")
		}
	}
	if withAnswers != 1 {
		t.Errorf("ListCompanyAnswers() = %d jobs with a current answer, want 1", withAnswers)
	}
}

func TestSaveScores(t *testing.T) {
	st, pool := newStore(t)
	userID := pgtest.InsertUser(t, pool)
	jobID := pgtest.InsertJob(t, pool, "Engineer", "fp-1")
	insertScore(t, pool, jobID, userID)

	if err := st.SaveScores(t.Context(), []dto.JobScore{{JobID: jobID, UserID: userID, Score: 90, Rows: []dto.ScoreRow{}}}); err != nil {
		t.Fatalf("SaveScores() err = %v", err)
	}
	got := countRows(t, pool, `SELECT suitability_score FROM job_scores WHERE job_id = $1 AND user_id = $2`, jobID, userID)
	if got != 90 {
		t.Errorf("suitability_score = %d, want 90", got)
	}
}

func insertTrackedCompany(t *testing.T, pool *pgxpool.Pool, slug string) (companyID, userID string) {
	t.Helper()
	if err := pool.QueryRow(t.Context(), `INSERT INTO companies (slug, name) VALUES ($1, $1) RETURNING id`, slug).Scan(&companyID); err != nil {
		t.Fatalf("insert company: %v", err)
	}
	userID = pgtest.InsertUser(t, pool)
	exec(t, pool, `INSERT INTO tracked_companies (user_id, company_id, enabled) VALUES ($1, $2, true)`, userID, companyID)
	return companyID, userID
}

func TestJobsChanged(t *testing.T) {
	tests := []struct {
		name        string
		commit      bool
		wantAnswers int
		wantEffects int
	}{
		{"commit drops stale answers and queues a first-discovery effect", true, 0, 1},
		{"rollback keeps the answers and queues nothing", false, 1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, pool := newStore(t)
			jobID := pgtest.InsertJob(t, pool, "Engineer", "fp-new")
			insertAnswer(t, pool, jobID, "fp-old", "hash-1", "typesafe/jev-1.13")
			insertTrackedCompany(t, pool, "acme")

			pgtest.InTx(t, pool, tt.commit, func(tx pgx.Tx) error {
				return st.JobsChanged(t.Context(), tx, []string{jobID}, true)
			})

			if got := countRows(t, pool, `SELECT count(*) FROM option_answers WHERE job_id = $1`, jobID); got != tt.wantAnswers {
				t.Errorf("answers = %d, want %d", got, tt.wantAnswers)
			}
			if got := countRows(t, pool, `SELECT count(*) FROM effect_outbox WHERE job_id = $1 AND first_discovery`, jobID); got != tt.wantEffects {
				t.Errorf("first-discovery effects = %d, want %d", got, tt.wantEffects)
			}
		})
	}
}

func TestJobsClosed(t *testing.T) {
	tests := []struct {
		name        string
		commit      bool
		wantAnswers int
	}{
		{"commit drops the answers", true, 0},
		{"rollback keeps the answers", false, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, pool := newStore(t)
			jobID := pgtest.InsertJob(t, pool, "Engineer", "fp-1")
			insertAnswer(t, pool, jobID, "fp-1", "hash-1", "typesafe/jev-1.13")

			pgtest.InTx(t, pool, tt.commit, func(tx pgx.Tx) error {
				return st.JobsClosed(t.Context(), tx, []string{jobID})
			})

			if got := countRows(t, pool, `SELECT count(*) FROM option_answers WHERE job_id = $1`, jobID); got != tt.wantAnswers {
				t.Errorf("answers = %d, want %d", got, tt.wantAnswers)
			}
		})
	}
}

func TestCompanyTracked(t *testing.T) {
	tests := []struct {
		name        string
		commit      bool
		wantEffects int
	}{
		{"commit queues a scoring effect without alerting", true, 1},
		{"rollback queues nothing", false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, pool := newStore(t)
			companyID, userID := insertTrackedCompany(t, pool, "tracked-co")
			var jobID string
			err := pool.QueryRow(t.Context(),
				`INSERT INTO jobs (title, location, url, company_slug, company_id, source, updated_at, content_fingerprint)
				 VALUES ('Engineer', 'Remote', 'https://example.com/tracked', 'tracked-co', $1, 'greenhouse', NOW(), 'fp-1') RETURNING id`,
				companyID).Scan(&jobID)
			if err != nil {
				t.Fatalf("insert job: %v", err)
			}

			pgtest.InTx(t, pool, tt.commit, func(tx pgx.Tx) error {
				return st.CompanyTracked(t.Context(), tx, userID, companyID)
			})

			if got := countRows(t, pool, `SELECT count(*) FROM effect_outbox WHERE job_id = $1 AND NOT first_discovery`, jobID); got != tt.wantEffects {
				t.Errorf("silent effects = %d, want %d", got, tt.wantEffects)
			}
			if got := countRows(t, pool, `SELECT count(*) FROM effect_outbox WHERE job_id = $1 AND first_discovery`, jobID); got != 0 {
				t.Errorf("alerting effects = %d, want 0", got)
			}
		})
	}
}

func insertEffectOutbox(t *testing.T, pool *pgxpool.Pool, jobID, status string, createdAt time.Time) {
	t.Helper()
	exec(t, pool,
		`INSERT INTO effect_outbox (job_id, fingerprint, status, created_at) VALUES ($1, 'fp', $2, $3)`,
		jobID, status, createdAt)
}

func insertBoard(t *testing.T, pool *pgxpool.Pool, status string, nextDueAt time.Time, leaseUntil *time.Time, failures int) {
	t.Helper()
	n := seedCounter.Add(1)
	var companyID, boardID string
	if err := pool.QueryRow(t.Context(),
		`INSERT INTO companies (slug, name) VALUES ($1, $1) RETURNING id`, fmt.Sprintf("co-%d", n)).Scan(&companyID); err != nil {
		t.Fatalf("insert company: %v", err)
	}
	if err := pool.QueryRow(t.Context(),
		`INSERT INTO company_boards (company_id, source, board_token, status, verification_method, verified_at) VALUES ($1, 'greenhouse', $2, $3, 'manual', NOW()) RETURNING id`,
		companyID, fmt.Sprintf("tok-%d", n), status).Scan(&boardID); err != nil {
		t.Fatalf("insert board: %v", err)
	}
	exec(t, pool,
		`INSERT INTO board_poll_state (board_id, next_due_at, lease_until, consecutive_failures) VALUES ($1, $2, $3, $4)`,
		boardID, nextDueAt, leaseUntil, failures)
}

func TestOpsState(t *testing.T) {
	t.Run("counts pending, running and failed effects", func(t *testing.T) {
		st, pool := newStore(t)
		now := time.Now()
		insertEffectOutbox(t, pool, pgtest.InsertJob(t, pool, "Engineer", "fp-1"), "pending", now.Add(-2*time.Hour))
		insertEffectOutbox(t, pool, pgtest.InsertJob(t, pool, "Engineer", "fp-2"), "running", now.Add(-10*time.Minute))
		insertEffectOutbox(t, pool, pgtest.InsertJob(t, pool, "Engineer", "fp-3"), "failed", now)
		insertEffectOutbox(t, pool, pgtest.InsertJob(t, pool, "Engineer", "fp-4"), "failed", now)
		insertEffectOutbox(t, pool, pgtest.InsertJob(t, pool, "Engineer", "fp-5"), "done", now)

		state := opsState(t, st)
		if state.OutboxPending != 2 {
			t.Errorf("OpsState().OutboxPending = %d, want %d", state.OutboxPending, 2)
		}
		if state.OutboxFailed != 2 {
			t.Errorf("OpsState().OutboxFailed = %d, want %d", state.OutboxFailed, 2)
		}
		if state.OutboxOldestPendingAge < 115*time.Minute || state.OutboxOldestPendingAge > 125*time.Minute {
			t.Errorf("OpsState().OutboxOldestPendingAge = %s, want ~2h", state.OutboxOldestPendingAge)
		}
	})

	t.Run("counts verified unleased boards past due as overdue", func(t *testing.T) {
		st, pool := newStore(t)
		now := time.Now()
		overdue := now.Add(-2 * time.Hour)
		liveLease := now.Add(time.Hour)
		expiredLease := now.Add(-time.Hour)
		insertBoard(t, pool, "verified", overdue, nil, 0)
		insertBoard(t, pool, "verified", overdue, &liveLease, 0)
		insertBoard(t, pool, "verified", overdue, &expiredLease, 0)
		insertBoard(t, pool, "retired", overdue, nil, 0)
		insertBoard(t, pool, "candidate", overdue, nil, 0)
		insertBoard(t, pool, "verified", now.Add(2*time.Hour), nil, 0)

		if got := opsState(t, st).BoardsOverdue; got != 2 {
			t.Errorf("OpsState().BoardsOverdue = %d, want %d", got, 2)
		}
	})

	t.Run("counts verified boards with three consecutive failures as failing", func(t *testing.T) {
		st, pool := newStore(t)
		due := time.Now().Add(time.Hour)
		insertBoard(t, pool, "verified", due, nil, 3)
		insertBoard(t, pool, "verified", due, nil, 2)
		insertBoard(t, pool, "retired", due, nil, 5)

		if got := opsState(t, st).BoardsFailing; got != 1 {
			t.Errorf("OpsState().BoardsFailing = %d, want %d", got, 1)
		}
	})

	t.Run("counts source targets whose last run failed", func(t *testing.T) {
		st, pool := newStore(t)
		userID := pgtest.InsertUser(t, pool)
		for i, status := range []string{"failed", "failed", "succeeded", "idle"} {
			exec(t, pool,
				`INSERT INTO source_targets (user_id, source, value, run_status) VALUES ($1, 'linkedin', $2, $3)`,
				userID, fmt.Sprintf("q-%d", i), status)
		}

		if got := opsState(t, st).SourceTargetsFailed; got != 2 {
			t.Errorf("OpsState().SourceTargetsFailed = %d, want %d", got, 2)
		}
	})

	t.Run("counts disabled source targets per source", func(t *testing.T) {
		st, pool := newStore(t)
		userID := pgtest.InsertUser(t, pool)
		for i, row := range []struct{ source, reason string }{{"indeed", "indeed key rejected"}, {"indeed", "indeed key rejected"}, {"indeed", ""}, {"linkedin", ""}} {
			exec(t, pool,
				`INSERT INTO source_targets (user_id, source, value, disabled_reason) VALUES ($1, $2, $3, $4)`,
				userID, row.source, fmt.Sprintf("q-%d", i), row.reason)
		}

		want := map[string]int64{"indeed": 2}
		if diff := cmp.Diff(want, opsState(t, st).DisabledSourceTargets); diff != "" {
			t.Errorf("OpsState().DisabledSourceTargets mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("counts unique relevant jobs per source", func(t *testing.T) {
		st, pool := newStore(t)
		userID := pgtest.InsertUser(t, pool)
		type u struct {
			source string
			age    time.Duration
		}
		seed := func(fp string, scored bool, urls ...u) {
			jobID := pgtest.InsertJob(t, pool, fp, fp)
			if scored {
				insertScore(t, pool, jobID, userID)
			}
			for i, u := range urls {
				exec(t, pool, `INSERT INTO job_urls (job_id, normalized_url, source, first_seen_at) VALUES ($1, $2, $3, $4)`,
					jobID, fmt.Sprintf("%s-%d", fp, i), u.source, time.Now().Add(-u.age))
			}
		}
		day := 24 * time.Hour
		seed("lever-a", true, u{"lever", day})
		seed("lever-b", true, u{"lever", 2 * day})
		seed("lever-old", true, u{"lever", 20 * day})
		seed("lever-unscored", false, u{"lever", day})
		seed("shared", true, u{"lever", day}, u{"ashby", day})
		seed("ashby-a", true, u{"ashby", day})

		want := map[string]int64{"lever": 2, "ashby": 1}
		if diff := cmp.Diff(want, opsState(t, st).UniqueRelevantJobs); diff != "" {
			t.Errorf("OpsState().UniqueRelevantJobs mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("reports harvest age per harvester", func(t *testing.T) {
		st, pool := newStore(t)
		now := time.Now()
		for harvester, at := range map[string]time.Time{"ashby": now.Add(-3 * time.Hour), "lever": now.Add(-time.Minute)} {
			exec(t, pool, `INSERT INTO harvest_runs (harvester, last_succeeded_at) VALUES ($1, $2)`, harvester, at)
		}

		state := opsState(t, st)
		if len(state.HarvestAge) != 2 {
			t.Fatalf("len(OpsState().HarvestAge) = %d, want %d", len(state.HarvestAge), 2)
		}
		if age := state.HarvestAge["ashby"]; age < 179*time.Minute || age > 181*time.Minute {
			t.Errorf("OpsState().HarvestAge[ashby] = %s, want ~3h", age)
		}
		if age := state.HarvestAge["lever"]; age < 0 || age > 2*time.Minute {
			t.Errorf("OpsState().HarvestAge[lever] = %s, want ~1m", age)
		}
	})

	t.Run("gauges boards, relevant jobs and admitted companies by discovery route", func(t *testing.T) {
		st, pool := newStore(t)
		userID := pgtest.InsertUser(t, pool)
		exec(t, pool, `INSERT INTO harvest_runs (harvester, last_succeeded_at) VALUES ('ashby', NOW())`)
		day := 24 * time.Hour
		seed := func(slug, via, status string, verifiedAgo time.Duration, reviewState string, scoredJobs int) {
			var companyID, boardID string
			if err := pool.QueryRow(t.Context(), `INSERT INTO companies (slug, name) VALUES ($1, $1) RETURNING id`, slug).Scan(&companyID); err != nil {
				t.Fatalf("insert company: %v", err)
			}
			if err := pool.QueryRow(t.Context(),
				`INSERT INTO company_boards (company_id, source, board_token, status, verification_method, verified_at, discovered_via)
				 VALUES ($1, 'greenhouse', $2, $3, 'manual', $4, NULLIF($5, '')) RETURNING id`,
				companyID, slug, status, time.Now().Add(-verifiedAgo), via).Scan(&boardID); err != nil {
				t.Fatalf("insert board: %v", err)
			}
			if reviewState != "" {
				exec(t, pool, `INSERT INTO tracked_companies (user_id, company_id, review_state) VALUES ($1, $2, $3)`, userID, companyID, reviewState)
			}
			for i := range scoredJobs {
				jobID := pgtest.InsertJob(t, pool, slug, fmt.Sprintf("%s-%d", slug, i))
				exec(t, pool, `UPDATE jobs SET primary_board_id = $1 WHERE id = $2`, boardID, jobID)
				insertScore(t, pool, jobID, userID)
			}
		}
		seed("li-a", "linkedin", "verified", day, "", 2)
		seed("li-b", "linkedin", "verified", 2*day, "", 1)
		seed("li-stale", "linkedin", "verified", 20*day, "", 1)
		seed("li-retired", "linkedin", "retired", day, "", 1)
		seed("ashby-kept", "ashby", "verified", day, "kept", 1)
		seed("ashby-new", "ashby", "verified", day, "new", 0)
		seed("ashby-dismissed", "ashby", "verified", day, "dismissed", 0)
		seed("linkedin-tracked", "linkedin", "verified", day, "kept", 0)
		seed("unattributed", "", "verified", day, "kept", 1)

		state := opsState(t, st)
		if diff := cmp.Diff(map[string]int64{"linkedin": 3, "ashby": 3}, state.DiscoveryBoards); diff != "" {
			t.Errorf("OpsState().DiscoveryBoards mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(map[string]int64{"linkedin": 3, "ashby": 1}, state.DiscoveryRelevantJobs); diff != "" {
			t.Errorf("OpsState().DiscoveryRelevantJobs mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(map[string]int64{"ashby": 2}, state.HarvestAdmitted); diff != "" {
			t.Errorf("OpsState().HarvestAdmitted mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestQueueMissingAnswers(t *testing.T) {
	const model = "typesafe/jev-1.13"

	t.Run("queues only open scored fingerprinted jobs missing the hash", func(t *testing.T) {
		st, pool := newStore(t)
		userA := pgtest.InsertUser(t, pool)
		userB := pgtest.InsertUser(t, pool)
		missing := pgtest.InsertJob(t, pool, "Engineer", "fp-missing")
		closed := pgtest.InsertJob(t, pool, "Engineer", "fp-closed")
		answered := pgtest.InsertJob(t, pool, "Engineer", "fp-answered")
		otherUsers := pgtest.InsertJob(t, pool, "Engineer", "fp-other-user")
		for _, jobID := range []string{missing, closed, answered} {
			insertScore(t, pool, jobID, userA)
		}
		insertScore(t, pool, otherUsers, userB)
		closeJob(t, pool, closed)
		insertAnswer(t, pool, answered, "fp-answered", "hash-1", model)

		n, err := st.QueueMissingAnswers(t.Context(), userA, []string{"hash-1"}, model)
		if err != nil {
			t.Fatalf("QueueMissingAnswers() err = %v", err)
		}
		if n != 1 {
			t.Errorf("QueueMissingAnswers() = %d, want 1", n)
		}
		for jobID, want := range map[string]int{missing: 1, closed: 0, answered: 0, otherUsers: 0} {
			if got := countRows(t, pool, "SELECT count(*) FROM effect_outbox WHERE job_id = $1", jobID); got != want {
				t.Errorf("effects for job %s = %d, want %d", jobID, got, want)
			}
		}
		if got := countRows(t, pool, "SELECT count(*) FROM effect_outbox WHERE first_discovery"); got != 0 {
			t.Errorf("first-discovery effects = %d, want 0 (no alert on backfill)", got)
		}
	})

	t.Run("quiesces when an effect is pending or no gap remains", func(t *testing.T) {
		st, pool := newStore(t)
		userID := pgtest.InsertUser(t, pool)
		jobID := pgtest.InsertJob(t, pool, "Engineer", "fp-1")
		insertScore(t, pool, jobID, userID)

		queue := func() int64 {
			n, err := st.QueueMissingAnswers(t.Context(), userID, []string{"hash-1"}, model)
			if err != nil {
				t.Fatalf("QueueMissingAnswers() err = %v", err)
			}
			return n
		}
		if got := queue(); got != 1 {
			t.Fatalf("first QueueMissingAnswers() = %d, want 1", got)
		}
		if got := queue(); got != 0 {
			t.Errorf("QueueMissingAnswers() while an effect is pending = %d, want 0", got)
		}

		exec(t, pool, "DELETE FROM effect_outbox WHERE job_id = $1", jobID)
		insertAnswer(t, pool, jobID, "fp-1", "hash-1", model)
		if got := queue(); got != 0 {
			t.Errorf("QueueMissingAnswers() with no gap = %d, want 0", got)
		}
	})
}

func TestClaimAnswerEffect_ConcurrentClaimsExactlyOneWinner(t *testing.T) {
	st, pool := newStore(t)
	jobID := pgtest.InsertJob(t, pool, "Engineer", "fp-1")
	insertEffect(t, pool, jobID, "fp-1")

	const claimers = 8
	var wins, notFounds atomic.Int64
	var wg sync.WaitGroup
	wg.Add(claimers)
	for range claimers {
		go func() {
			defer wg.Done()
			_, err := st.ClaimAnswerEffect(t.Context())
			switch {
			case err == nil:
				wins.Add(1)
			case errors.Is(err, data.ErrNotFound):
				notFounds.Add(1)
			default:
				t.Errorf("ClaimAnswerEffect() err = %v", err)
			}
		}()
	}
	wg.Wait()

	if wins.Load() != 1 || notFounds.Load() != claimers-1 {
		t.Errorf("winners = %d, losers = %d, want 1 and %d", wins.Load(), notFounds.Load(), claimers-1)
	}
}

func TestSaveAnswers_KeepsExistingAndRoundTrips(t *testing.T) {
	st, pool := newStore(t)
	ctx := t.Context()
	jobID := pgtest.InsertJob(t, pool, "Engineer", "fp-1")

	if err := st.SaveAnswers(ctx, jobID, "fp-1", "m", map[string]dto.Answer{"h1": {PYes: 0.5}}); err != nil {
		t.Fatalf("SaveAnswers() err = %v", err)
	}
	if err := st.SaveAnswers(ctx, jobID, "fp-1", "m", map[string]dto.Answer{"h1": {PYes: 0.9}, "h2": {PNo: 0.5}}); err != nil {
		t.Fatalf("SaveAnswers() err = %v", err)
	}
	got, err := st.ListAnswers(ctx, jobID, "fp-1", "m")
	if err != nil {
		t.Fatalf("ListAnswers() err = %v", err)
	}
	want := map[string]dto.Answer{"h1": {PYes: 0.5}, "h2": {PNo: 0.5}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ListAnswers() (-want +got):\n%s", diff)
	}
}

func TestPushSubscriptions(t *testing.T) {
	st, pool := newStore(t)
	ctx := t.Context()
	userID := pgtest.InsertUser(t, pool)
	sub := dto.PushSubscriptionInput{Endpoint: "https://push.example/a", Keys: dto.PushKeys{P256dh: "p1", Auth: "a1"}}

	if err := st.UpsertPushSubscription(ctx, userID, sub); err != nil {
		t.Fatalf("UpsertPushSubscription() err = %v", err)
	}
	sub.Keys = dto.PushKeys{P256dh: "p2", Auth: "a2"}
	if err := st.UpsertPushSubscription(ctx, userID, sub); err != nil {
		t.Fatalf("UpsertPushSubscription() again err = %v", err)
	}

	got, err := st.ListPushSubscriptions(ctx, userID)
	if err != nil {
		t.Fatalf("ListPushSubscriptions() err = %v", err)
	}
	if diff := cmp.Diff([]dto.PushSubscriptionInput{sub}, got); diff != "" {
		t.Errorf("ListPushSubscriptions() after re-upsert (-want +got):\n%s", diff)
	}

	for range 2 {
		if err := st.DeletePushSubscription(ctx, userID, sub.Endpoint); err != nil {
			t.Fatalf("DeletePushSubscription() err = %v", err)
		}
	}
	got, err = st.ListPushSubscriptions(ctx, userID)
	if err != nil || len(got) != 0 {
		t.Errorf("ListPushSubscriptions() after delete = %v, %v, want empty", got, err)
	}
}

func TestScoreFeedback_BlankReasonIsRejectedAndWritesNothing(t *testing.T) {
	st, pool := newStore(t)
	userID := pgtest.InsertUser(t, pool)

	_, err := st.InsertScoreFeedback(t.Context(), userID, dto.ScoreFeedback{Kind: "overall", Reason: "  ", Picks: []dto.Pick{}, Model: "m"})
	if err == nil {
		t.Fatal("InsertScoreFeedback(blank reason) err = nil, want the CHECK to reject it")
	}
	if n := countRows(t, pool, "SELECT count(*) FROM score_feedback"); n != 0 {
		t.Errorf("score_feedback rows = %d, want 0", n)
	}
}

func TestScoreFeedback_InsertRoundTripsPicksAndModel(t *testing.T) {
	st, pool := newStore(t)
	userID := pgtest.InsertUser(t, pool)
	picks := []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "manual"}}

	if _, err := st.InsertScoreFeedback(t.Context(), userID, dto.ScoreFeedback{Kind: "overall", Reason: "too generous", Picks: picks, Model: "jev-1"}); err != nil {
		t.Fatalf("InsertScoreFeedback() err = %v", err)
	}

	got, err := st.ListScoreFeedback(t.Context(), userID, dto.ScoreFeedbackFilter{Model: "m", IncludeOutdated: true}, 10, 0)
	if err != nil || len(got) != 1 {
		t.Fatalf("ListScoreFeedback() = %+v, %v, want one entry", got, err)
	}
	if diff := cmp.Diff(picks, got[0].Picks); diff != "" {
		t.Errorf("ListScoreFeedback() picks (-want +got):\n%s", diff)
	}
	if got[0].Kind != "overall" || got[0].Model != "jev-1" {
		t.Errorf("ListScoreFeedback() = %+v, want kind overall and model jev-1", got[0])
	}
	if n := countRows(t, pool, "SELECT count(*) FROM score_feedback WHERE job_id IS NULL AND direction IS NULL"); n != 1 {
		t.Errorf("rows with NULL job_id and direction = %d, want 1", n)
	}
}

func TestScoreFeedback_JobEntryRoundTripsAndSurvivesJobDelete(t *testing.T) {
	st, pool := newStore(t)
	userID := pgtest.InsertUser(t, pool)
	jobID := pgtest.InsertJob(t, pool, "Engineer", "fp-1")
	direction := "lower"
	score := 50

	entry := dto.ScoreFeedback{
		Kind: "job", Direction: &direction, JobID: &jobID, Reason: "r", Picks: []dto.Pick{}, Model: "m",
		Snapshot: dto.ScoreFeedbackSnapshot{Score: &score, ContentFingerprint: "fp-1"},
	}
	if _, err := st.InsertScoreFeedback(t.Context(), userID, entry); err != nil {
		t.Fatalf("InsertScoreFeedback() err = %v", err)
	}
	got, err := st.ListScoreFeedback(t.Context(), userID, dto.ScoreFeedbackFilter{Model: "m", IncludeOutdated: true}, 10, 0)
	if err != nil || len(got) != 1 {
		t.Fatalf("ListScoreFeedback() = %+v, %v, want one entry", got, err)
	}
	if diff := cmp.Diff(entry, got[0], cmpopts.IgnoreFields(dto.ScoreFeedback{}, "ID", "CreatedAt")); diff != "" {
		t.Errorf("ListScoreFeedback() (-want +got):\n%s", diff)
	}

	exec(t, pool, `DELETE FROM jobs WHERE id = $1`, jobID)
	got, err = st.ListScoreFeedback(t.Context(), userID, dto.ScoreFeedbackFilter{Model: "m", IncludeOutdated: true}, 10, 0)
	if err != nil || len(got) != 1 || got[0].JobID != nil {
		t.Errorf("ListScoreFeedback() after job delete = %+v, %v, want the entry kept with no job id", got, err)
	}
}

func TestScoreFeedback_DirectionWithoutJobKindIsRejected(t *testing.T) {
	st, pool := newStore(t)
	userID := pgtest.InsertUser(t, pool)
	direction := "lower"

	_, err := st.InsertScoreFeedback(t.Context(), userID, dto.ScoreFeedback{Kind: "overall", Direction: &direction, Reason: "r", Picks: []dto.Pick{}, Model: "m"})
	if err == nil {
		t.Error("InsertScoreFeedback(overall with direction) err = nil, want the kind/direction CHECK to reject it")
	}
}

func TestGetJobScoreForFeedback(t *testing.T) {
	st, pool := newStore(t)
	userID := pgtest.InsertUser(t, pool)
	jobID := pgtest.InsertJob(t, pool, "Engineer", "fp-1")
	exec(t, pool, `INSERT INTO job_scores (job_id, user_id, suitability_score, breakdown, score_fingerprint, score_model)
		VALUES ($1, $2, 64, '[{"key":"tech:go","label":"Go","stance":"nice","resolved":"yes","effect":"meets","overridden":false}]', 'fp-1', 'jev-1')`, jobID, userID)

	got, err := st.GetJobScoreForFeedback(t.Context(), userID, jobID)
	if err != nil {
		t.Fatalf("GetJobScoreForFeedback() err = %v", err)
	}
	want := dto.JobScoreEvidence{
		Score: 64, Fingerprint: "fp-1", Model: "jev-1",
		Breakdown: []dto.ScoreRow{{Key: "tech:go", Label: "Go", Stance: "nice", Resolved: "yes", Effect: "meets"}},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GetJobScoreForFeedback() (-want +got):\n%s", diff)
	}
}

func TestListJobScoresForCollection(t *testing.T) {
	st, pool := newStore(t)
	userID, otherID := pgtest.InsertUser(t, pool), pgtest.InsertUser(t, pool)
	scored := pgtest.InsertJob(t, pool, "Scored", "fp-1")
	unscored := pgtest.InsertJob(t, pool, "Unscored", "fp-2")
	exec(t, pool, `INSERT INTO job_scores (job_id, user_id, suitability_score, breakdown) VALUES
		($1, $2, 64, '[{"key":"tech:go","label":"Go","stance":"nice","resolved":"yes","effect":"meets","overridden":false}]'),
		($1, $3, 10, '[]')`, scored, userID, otherID)

	got, err := st.ListJobScoresForCollection(t.Context(), userID, []string{scored, unscored, "00000000-0000-0000-0000-000000000000"})
	if err != nil {
		t.Fatalf("ListJobScoresForCollection() err = %v", err)
	}
	if _, err := st.ListJobScoresForCollection(t.Context(), userID, []string{"not-a-uuid"}); !apperr.IsKind(err, apperr.KindInvalid) {
		t.Errorf("ListJobScoresForCollection(malformed id) err = %v, want Invalid", err)
	}
	score := 64
	want := []dto.CollectionJobScore{
		{JobID: scored, Title: "Scored", Company: "acme", Score: &score, Breakdown: []dto.ScoreRow{{Key: "tech:go", Label: "Go", Stance: "nice", Resolved: "yes", Effect: "meets"}}},
		{JobID: unscored, Title: "Unscored", Company: "acme", Breakdown: []dto.ScoreRow{}},
	}
	if diff := cmp.Diff(want, got, cmpopts.SortSlices(func(a, b dto.CollectionJobScore) bool { return a.Title < b.Title })); diff != "" {
		t.Errorf("ListJobScoresForCollection() (-want +got):\n%s", diff)
	}
}

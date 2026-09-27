package store_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/store"
)

var seedCounter atomic.Int64

func newStore(t *testing.T) (*store.Store, *pgxpool.Pool) {
	t.Helper()
	pool := pgtest.New(t)
	return store.New(pool), pool
}

func insertUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	username := fmt.Sprintf("user-%s-%d", t.Name(), seedCounter.Add(1))
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (username, password_hash) VALUES ($1, 'hash') RETURNING id`, username).Scan(&id)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return id
}

func insertJob(t *testing.T, pool *pgxpool.Pool, fingerprint string) string {
	t.Helper()
	var id string
	url := fmt.Sprintf("https://example.com/%s/%d", t.Name(), seedCounter.Add(1))
	err := pool.QueryRow(context.Background(),
		`INSERT INTO jobs (title, location, url, company_slug, source, updated_at, content_fingerprint)
		 VALUES ('Engineer', 'Remote', $1, 'acme', 'greenhouse', NOW(), $2) RETURNING id`,
		url, fingerprint).Scan(&id)
	if err != nil {
		t.Fatalf("insert job: %v", err)
	}
	return id
}

func insertEffect(t *testing.T, pool *pgxpool.Pool, jobID, fingerprint string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO effect_outbox (job_id, fingerprint) VALUES ($1, $2) RETURNING id`,
		jobID, fingerprint).Scan(&id)
	if err != nil {
		t.Fatalf("insert effect: %v", err)
	}
	return id
}

func TestSearchConfig_UpsertThenGetRoundTrips(t *testing.T) {
	st, pool := newStore(t)
	userID := insertUser(t, pool)

	saved, err := st.UpsertSearchConfig(context.Background(), dto.SearchConfig{
		UserID: userID, NotifyThreshold: 70,
		Preferences: dto.Preferences{Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "manual"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.NotifyThreshold != 70 || len(saved.Preferences.Picks) != 1 {
		t.Fatalf("saved = %+v, want threshold 70 with one pick", saved)
	}

	got, err := st.GetSearchConfig(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if got.UserID != userID || got.NotifyThreshold != 70 {
		t.Fatalf("got = %+v, want userID %q with threshold 70", got, userID)
	}
}

func TestGetSearchConfig_MissingReturnsErrNotFound(t *testing.T) {
	st, pool := newStore(t)
	userID := insertUser(t, pool)

	_, err := st.GetSearchConfig(context.Background(), userID)
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestScoringOptions_AddRewordRetireLifecycle(t *testing.T) {
	st, _ := newStore(t)
	ctx := context.Background()

	if err := st.AddScoringOption(ctx, "tech:go", "tech", "Go", "Does the role use Go?"); err != nil {
		t.Fatalf("AddScoringOption: %v", err)
	}
	options, err := st.ListScoringOptions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(options) != 1 || options[0].Question != "Does the role use Go?" {
		t.Fatalf("options = %+v, want one Go option", options)
	}

	if err := st.RewordScoringOption(ctx, "tech:go", "Does the role primarily use Go?"); err != nil {
		t.Fatalf("RewordScoringOption: %v", err)
	}
	options, err = st.ListScoringOptions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if options[0].Question != "Does the role primarily use Go?" {
		t.Fatalf("question = %q, want reworded", options[0].Question)
	}

	if err := st.RetireScoringOption(ctx, "tech:go"); err != nil {
		t.Fatalf("RetireScoringOption: %v", err)
	}
	options, err = st.ListScoringOptions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if options[0].RetiredAt == nil {
		t.Fatalf("options = %+v, want retired_at set", options)
	}
}

func TestRewordScoringOption_MissingReturnsErrNotFound(t *testing.T) {
	st, _ := newStore(t)
	err := st.RewordScoringOption(context.Background(), "missing", "question")
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestRetireScoringOption_MissingReturnsErrNotFound(t *testing.T) {
	st, _ := newStore(t)
	err := st.RetireScoringOption(context.Background(), "missing")
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestRetireScoringOption_AlreadyRetiredReturnsErrNotFound(t *testing.T) {
	st, _ := newStore(t)
	ctx := context.Background()
	if err := st.AddScoringOption(ctx, "tech:zig", "tech", "Zig", "Does the role use Zig?"); err != nil {
		t.Fatal(err)
	}
	if err := st.RetireScoringOption(ctx, "tech:zig"); err != nil {
		t.Fatal(err)
	}
	if err := st.RetireScoringOption(ctx, "tech:zig"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestAddScoringOption_QueuesBackfillForScoredOpenJobsOnly(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()

	userID := insertUser(t, pool)
	scored := insertJob(t, pool, "fp-scored")
	closed := insertJob(t, pool, "fp-closed")
	unscored := insertJob(t, pool, "fp-unscored")
	for _, jobID := range []string{scored, closed} {
		if _, err := pool.Exec(ctx, "INSERT INTO job_scores (job_id, user_id) VALUES ($1, $2)", jobID, userID); err != nil {
			t.Fatalf("seed job_scores: %v", err)
		}
	}
	if _, err := pool.Exec(ctx, "UPDATE jobs SET closed_at = NOW() WHERE id = $1", closed); err != nil {
		t.Fatalf("close job: %v", err)
	}

	if err := st.AddScoringOption(ctx, "tech:zig", "tech", "Zig", "Does the role use Zig?"); err != nil {
		t.Fatalf("AddScoringOption: %v", err)
	}

	for jobID, want := range map[string]int{scored: 1, closed: 0, unscored: 0} {
		var count int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM effect_outbox WHERE job_id = $1", jobID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Errorf("effects for job %s = %d, want %d", jobID, count, want)
		}
	}
}

func TestListInterestedConfigs_JoinsTrackedCompany(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	userID := insertUser(t, pool)
	jobID := insertJob(t, pool, "fp-1")

	var companyID string
	if err := pool.QueryRow(ctx, `INSERT INTO companies (slug, name) VALUES ('acme', 'Acme') RETURNING id`).Scan(&companyID); err != nil {
		t.Fatalf("insert company: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO tracked_companies (user_id, company_id, enabled) VALUES ($1, $2, true)`, userID, companyID); err != nil {
		t.Fatalf("insert tracked company: %v", err)
	}

	configs, err := st.ListInterestedConfigs(ctx, jobID, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(configs) != 1 || configs[0].UserID != userID {
		t.Fatalf("configs = %+v, want one config for %q", configs, userID)
	}
}

func TestClaimAnswerEffect_ThenCompleteWritesAnswersAndScores(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	userID := insertUser(t, pool)
	jobID := insertJob(t, pool, "fp-1")
	insertEffect(t, pool, jobID, "fp-1")

	effect, err := st.ClaimAnswerEffect(ctx)
	if err != nil {
		t.Fatalf("ClaimAnswerEffect: %v", err)
	}
	if effect.JobID != jobID || effect.Attempts != 1 {
		t.Fatalf("effect = %+v, want jobID %q with attempts 1", effect, jobID)
	}

	saved, err := st.CompleteAnswerEffect(ctx, effect,
		map[string]dto.Answer{"hash-1": {PYes: 0.9, PNo: 0.05, PNotStated: 0.05}},
		[]dto.JobScore{{JobID: jobID, UserID: userID, Score: 80, Rows: []dto.ScoreRow{}}},
	)
	if err != nil {
		t.Fatalf("CompleteAnswerEffect: %v", err)
	}
	if len(saved) != 1 || saved[0] != userID {
		t.Fatalf("saved = %v, want [%q]", saved, userID)
	}

	answers, err := st.ListAnswers(ctx, jobID, "fp-1", "typesafe/jev-1.13")
	if err != nil {
		t.Fatal(err)
	}
	if len(answers) != 1 || answers["hash-1"].PYes != float64(float32(0.9)) {
		t.Fatalf("answers = %+v, want one cached answer", answers)
	}

	status, err := st.GetScoringStatus(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if status.Pending != 0 {
		t.Fatalf("pending = %d, want 0 after completion", status.Pending)
	}
}

func TestCompleteAnswerEffect_CommitsBothUsersScoresTogether(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	alice := insertUser(t, pool)
	bob := insertUser(t, pool)
	jobID := insertJob(t, pool, "fp-1")
	insertEffect(t, pool, jobID, "fp-1")

	effect, err := st.ClaimAnswerEffect(ctx)
	if err != nil {
		t.Fatal(err)
	}

	scores := []dto.JobScore{
		{JobID: jobID, UserID: alice, Score: 79, Rows: []dto.ScoreRow{{Key: "tech:go", Stance: "nice", Resolved: "yes", Effect: "meets"}}},
		{JobID: jobID, UserID: bob, Score: 21, Rows: []dto.ScoreRow{{Key: "tech:go", Stance: "avoid", Resolved: "yes", Effect: "misses"}}},
	}
	saved, err := st.CompleteAnswerEffect(ctx, effect, map[string]dto.Answer{"hash-go": {PYes: 0.9, PNo: 0.05, PNotStated: 0.05}}, scores)
	if err != nil {
		t.Fatalf("CompleteAnswerEffect: %v", err)
	}
	if len(saved) != 2 {
		t.Fatalf("saved users = %v, want 2", saved)
	}

	var scoreCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM job_scores WHERE job_id = $1", jobID).Scan(&scoreCount); err != nil {
		t.Fatal(err)
	}
	if scoreCount != 2 {
		t.Fatalf("job_scores rows = %d, want 2", scoreCount)
	}
}

func TestCompleteAnswerEffect_FingerprintMismatchWritesNothing(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	userID := insertUser(t, pool)
	jobID := insertJob(t, pool, "fp-1")
	insertEffect(t, pool, jobID, "fp-1")

	effect, err := st.ClaimAnswerEffect(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(ctx, "UPDATE jobs SET content_fingerprint = 'fp-2' WHERE id = $1", jobID); err != nil {
		t.Fatalf("change fingerprint: %v", err)
	}

	saved, err := st.CompleteAnswerEffect(ctx, effect,
		map[string]dto.Answer{"hash-go": {PYes: 0.9, PNo: 0.05, PNotStated: 0.05}},
		[]dto.JobScore{{JobID: jobID, UserID: userID, Score: 79}},
	)
	if err != nil {
		t.Fatalf("CompleteAnswerEffect: %v", err)
	}
	if len(saved) != 0 {
		t.Fatalf("saved users = %v, want none (fingerprint moved on)", saved)
	}

	var answerCount, scoreCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM option_answers WHERE job_id = $1", jobID).Scan(&answerCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM job_scores WHERE job_id = $1", jobID).Scan(&scoreCount); err != nil {
		t.Fatal(err)
	}
	if answerCount != 0 || scoreCount != 0 {
		t.Fatalf("answers = %d, scores = %d, want 0 and 0", answerCount, scoreCount)
	}

	var status string
	if err := pool.QueryRow(ctx, "SELECT status FROM effect_outbox WHERE id = $1", effect.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "done" {
		t.Fatalf("effect status = %q, want done (a stale effect is still marked done)", status)
	}
}

func TestClaimAnswerEffect_EmptyQueueReturnsErrNotFound(t *testing.T) {
	st, _ := newStore(t)
	_, err := st.ClaimAnswerEffect(context.Background())
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestFailAnswerEffect_ReschedulesForRetry(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	jobID := insertJob(t, pool, "fp-1")
	insertEffect(t, pool, jobID, "fp-1")

	effect, err := st.ClaimAnswerEffect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.FailAnswerEffect(ctx, effect.ID, effect.Attempts, dto.ScoringFailure{Reason: "boom"}); err != nil {
		t.Fatalf("FailAnswerEffect: %v", err)
	}

	var status, lastError string
	err = pool.QueryRow(ctx, `SELECT status, last_error FROM effect_outbox WHERE id = $1`, effect.ID).Scan(&status, &lastError)
	if err != nil {
		t.Fatal(err)
	}
	if status != "pending" || lastError != "boom" {
		t.Fatalf("status = %q, lastError = %q, want pending/boom", status, lastError)
	}
}

func TestListScoringInputsThenSaveScores(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	userID := insertUser(t, pool)
	jobID := insertJob(t, pool, "fp-1")

	if _, err := pool.Exec(ctx,
		`INSERT INTO job_scores (job_id, user_id, suitability_score, breakdown) VALUES ($1, $2, 50, '[]')`,
		jobID, userID); err != nil {
		t.Fatalf("seed job score: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO option_answers (job_id, fingerprint, question_hash, model, p_yes, p_no, p_not_stated, confidence)
		 VALUES ($1, 'fp-1', 'hash-1', 'typesafe/jev-1.13', 0.9, 0.05, 0.05, 0.9)`, jobID); err != nil {
		t.Fatalf("seed answer: %v", err)
	}

	inputs, err := st.ListScoringInputs(ctx, userID, "typesafe/jev-1.13")
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) != 1 || inputs[0].Job.ID != jobID || len(inputs[0].Answers) != 1 {
		t.Fatalf("inputs = %+v, want one job with one answer", inputs)
	}

	if err := st.SaveScores(ctx, []dto.JobScore{{JobID: jobID, UserID: userID, Score: 90, Rows: []dto.ScoreRow{}}}); err != nil {
		t.Fatalf("SaveScores: %v", err)
	}
	var score int
	if err := pool.QueryRow(ctx, `SELECT suitability_score FROM job_scores WHERE job_id = $1 AND user_id = $2`, jobID, userID).Scan(&score); err != nil {
		t.Fatal(err)
	}
	if score != 90 {
		t.Fatalf("score = %d, want 90", score)
	}
}

func TestListScoringInputs_IgnoresAnswersUnderAnotherModel(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	userID := insertUser(t, pool)
	jobID := insertJob(t, pool, "fp-1")

	if _, err := pool.Exec(ctx,
		`INSERT INTO job_scores (job_id, user_id, suitability_score, breakdown) VALUES ($1, $2, 50, '[]')`,
		jobID, userID); err != nil {
		t.Fatalf("seed job score: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO option_answers (job_id, fingerprint, question_hash, model, p_yes, p_no, p_not_stated, confidence)
		 VALUES ($1, 'fp-1', 'hash-1', 'old-model', 0.9, 0.05, 0.05, 0.9)`, jobID); err != nil {
		t.Fatalf("seed answer: %v", err)
	}

	inputs, err := st.ListScoringInputs(ctx, userID, "typesafe/jev-1.13")
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) != 1 || len(inputs[0].Answers) != 0 {
		t.Fatalf("inputs = %+v, want one job with no answers (answer stored under another model)", inputs)
	}
}

func insertTrackedCompany(t *testing.T, pool *pgxpool.Pool, slug string) (companyID, userID string) {
	t.Helper()
	ctx := context.Background()
	if err := pool.QueryRow(ctx, `INSERT INTO companies (slug, name) VALUES ($1, $1) RETURNING id`, slug).Scan(&companyID); err != nil {
		t.Fatalf("insert company: %v", err)
	}
	userID = insertUser(t, pool)
	if _, err := pool.Exec(ctx, `INSERT INTO tracked_companies (user_id, company_id, enabled) VALUES ($1, $2, true)`, userID, companyID); err != nil {
		t.Fatalf("insert tracked company: %v", err)
	}
	return companyID, userID
}

func TestJobsChanged_DropsStaleAnswersAndQueuesEffect(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	jobID := insertJob(t, pool, "fp-new")
	if _, err := pool.Exec(ctx,
		`INSERT INTO option_answers (job_id, fingerprint, question_hash, model, p_yes, p_no, p_not_stated, confidence)
		 VALUES ($1, 'fp-old', 'hash-1', 'typesafe/jev-1.13', 0.9, 0.05, 0.05, 0.9)`, jobID); err != nil {
		t.Fatalf("seed stale answer: %v", err)
	}
	insertTrackedCompany(t, pool, "acme")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.JobsChanged(ctx, tx, []string{jobID}, true); err != nil {
		t.Fatalf("JobsChanged: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	var staleCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM option_answers WHERE job_id = $1`, jobID).Scan(&staleCount); err != nil {
		t.Fatal(err)
	}
	if staleCount != 0 {
		t.Fatalf("stale answers remaining = %d, want 0", staleCount)
	}

	var firstDiscovery bool
	if err := pool.QueryRow(ctx, `SELECT first_discovery FROM effect_outbox WHERE job_id = $1`, jobID).Scan(&firstDiscovery); err != nil {
		t.Fatalf("queued effect: %v", err)
	}
	if !firstDiscovery {
		t.Fatal("first_discovery = false, want true")
	}
}

func TestJobsChanged_RollbackAlsoRollsBackQueuedEffect(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	jobID := insertJob(t, pool, "fp-1")
	insertTrackedCompany(t, pool, "acme")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.JobsChanged(ctx, tx, []string{jobID}, true); err != nil {
		t.Fatalf("JobsChanged: %v", err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM effect_outbox WHERE job_id = $1`, jobID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("queued effects after rollback = %d, want 0: JobsChanged must run in the caller's own tx", count)
	}
}

func TestJobsClosed_DropsAnswers(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	jobID := insertJob(t, pool, "fp-1")
	if _, err := pool.Exec(ctx,
		`INSERT INTO option_answers (job_id, fingerprint, question_hash, model, p_yes, p_no, p_not_stated, confidence)
		 VALUES ($1, 'fp-1', 'hash-1', 'typesafe/jev-1.13', 0.9, 0.05, 0.05, 0.9)`, jobID); err != nil {
		t.Fatalf("seed answer: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.JobsClosed(ctx, tx, []string{jobID}); err != nil {
		t.Fatalf("JobsClosed: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM option_answers WHERE job_id = $1`, jobID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("answers remaining = %d, want 0", count)
	}
}

func TestCompanyTracked_QueuesScoresWithoutAlert(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	companyID, userID := insertTrackedCompany(t, pool, "tracked-co")
	var jobID string
	err := pool.QueryRow(ctx,
		`INSERT INTO jobs (title, location, url, company_slug, company_id, source, updated_at, content_fingerprint)
		 VALUES ('Engineer', 'Remote', 'https://example.com/tracked-co/1', 'tracked-co', $1, 'greenhouse', NOW(), 'fp-1') RETURNING id`,
		companyID).Scan(&jobID)
	if err != nil {
		t.Fatalf("insert job: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CompanyTracked(ctx, tx, userID, companyID); err != nil {
		t.Fatalf("CompanyTracked: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	var firstDiscovery bool
	if err := pool.QueryRow(ctx, `SELECT first_discovery FROM effect_outbox WHERE job_id = $1`, jobID).Scan(&firstDiscovery); err != nil {
		t.Fatalf("queued effect: %v", err)
	}
	if firstDiscovery {
		t.Fatal("first_discovery = true, want false: tracking must not alert")
	}
}

func insertEffectOutbox(t *testing.T, pool *pgxpool.Pool, jobID, status string, createdAt time.Time) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO effect_outbox (job_id, fingerprint, status, created_at) VALUES ($1, 'fp', $2, $3)`,
		jobID, status, createdAt); err != nil {
		t.Fatalf("insert effect_outbox (%s): %v", status, err)
	}
}

func TestOpsState_CountsPendingRunningAndFailed(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()

	now := time.Now()
	insertEffectOutbox(t, pool, insertJob(t, pool, "fp-1"), "pending", now.Add(-2*time.Hour))
	insertEffectOutbox(t, pool, insertJob(t, pool, "fp-2"), "running", now.Add(-10*time.Minute))
	insertEffectOutbox(t, pool, insertJob(t, pool, "fp-3"), "failed", now)
	insertEffectOutbox(t, pool, insertJob(t, pool, "fp-4"), "failed", now)
	insertEffectOutbox(t, pool, insertJob(t, pool, "fp-5"), "done", now)

	state, err := st.OpsState(ctx)
	if err != nil {
		t.Fatalf("OpsState: %v", err)
	}
	if state.OutboxPending != 2 {
		t.Errorf("OutboxPending = %d, want 2", state.OutboxPending)
	}
	if state.OutboxFailed != 2 {
		t.Errorf("OutboxFailed = %d, want 2", state.OutboxFailed)
	}
	if state.OutboxOldestPendingAge < 115*time.Minute || state.OutboxOldestPendingAge > 125*time.Minute {
		t.Errorf("OutboxOldestPendingAge = %s, want ~2h", state.OutboxOldestPendingAge)
	}
}

func TestOpsState_NoPendingIsZero(t *testing.T) {
	st, _ := newStore(t)
	state, err := st.OpsState(context.Background())
	if err != nil {
		t.Fatalf("OpsState: %v", err)
	}
	if state.OutboxPending != 0 || state.OutboxFailed != 0 {
		t.Errorf("expected zero counts, got %+v", state)
	}
	if state.OutboxOldestPendingAge != 0 {
		t.Errorf("OutboxOldestPendingAge = %s, want 0", state.OutboxOldestPendingAge)
	}
}

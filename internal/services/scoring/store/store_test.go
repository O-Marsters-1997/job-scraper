package store_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/scoringtest"
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

func TestScoringStoreContract(t *testing.T) {
	scoringtest.RunStoreContract(t, func(t *testing.T) scoring.Store {
		t.Helper()
		st, _ := newStore(t)
		return st
	})
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
	if !errors.Is(err, data.ErrNotFound) {
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
	if !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestRetireScoringOption_MissingReturnsErrNotFound(t *testing.T) {
	st, _ := newStore(t)
	err := st.RetireScoringOption(context.Background(), "missing")
	if !errors.Is(err, data.ErrNotFound) {
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
	if err := st.RetireScoringOption(ctx, "tech:zig"); !errors.Is(err, data.ErrNotFound) {
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
	if !errors.Is(err, data.ErrNotFound) {
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

func insertJobForCompany(t *testing.T, pool *pgxpool.Pool, companySlug, companyID string) string {
	t.Helper()
	var jobID string
	url := fmt.Sprintf("https://example.com/%s/%d", t.Name(), seedCounter.Add(1))
	err := pool.QueryRow(context.Background(),
		`INSERT INTO jobs (title, location, url, company_slug, company_id, source, updated_at, content_fingerprint)
		 VALUES ('Engineer', 'Remote', $1, $2, $3, 'greenhouse', NOW(), 'fp-1') RETURNING id`,
		url, companySlug, companyID).Scan(&jobID)
	if err != nil {
		t.Fatalf("insert job: %v", err)
	}
	return jobID
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

func TestJobsClosed_RollbackKeepsAnswers(t *testing.T) {
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
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM option_answers WHERE job_id = $1`, jobID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("answers remaining after rollback = %d, want 1: JobsClosed must run in the caller's own tx", count)
	}
}

func TestCompanyTracked_QueuesScoresWithoutAlert(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	companyID, userID := insertTrackedCompany(t, pool, "tracked-co")
	jobID := insertJobForCompany(t, pool, "tracked-co", companyID)

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

func TestCompanyTracked_RollbackQueuesNothing(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	companyID, userID := insertTrackedCompany(t, pool, "tracked-co")
	jobID := insertJobForCompany(t, pool, "tracked-co", companyID)

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CompanyTracked(ctx, tx, userID, companyID); err != nil {
		t.Fatalf("CompanyTracked: %v", err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM effect_outbox WHERE job_id = $1`, jobID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("queued effects after rollback = %d, want 0: CompanyTracked must run in the caller's own tx", count)
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

func insertBoard(t *testing.T, pool *pgxpool.Pool, status string, nextDueAt time.Time, leaseUntil *time.Time, failures int) {
	t.Helper()
	ctx := context.Background()
	n := seedCounter.Add(1)
	var companyID, boardID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO companies (slug, name) VALUES ($1, $1) RETURNING id`, fmt.Sprintf("co-%d", n)).Scan(&companyID); err != nil {
		t.Fatalf("insert company: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO company_boards (company_id, source, board_token, status, verification_method, verified_at) VALUES ($1, 'greenhouse', $2, $3, 'manual', NOW()) RETURNING id`,
		companyID, fmt.Sprintf("tok-%d", n), status).Scan(&boardID); err != nil {
		t.Fatalf("insert board: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO board_poll_state (board_id, next_due_at, lease_until, consecutive_failures) VALUES ($1, $2, $3, $4)`,
		boardID, nextDueAt, leaseUntil, failures); err != nil {
		t.Fatalf("insert board_poll_state: %v", err)
	}
}

func TestOpsState(t *testing.T) {
	t.Run("counts pending, running and failed effects", func(t *testing.T) {
		st, pool := newStore(t)
		now := time.Now()
		insertEffectOutbox(t, pool, insertJob(t, pool, "fp-1"), "pending", now.Add(-2*time.Hour))
		insertEffectOutbox(t, pool, insertJob(t, pool, "fp-2"), "running", now.Add(-10*time.Minute))
		insertEffectOutbox(t, pool, insertJob(t, pool, "fp-3"), "failed", now)
		insertEffectOutbox(t, pool, insertJob(t, pool, "fp-4"), "failed", now)
		insertEffectOutbox(t, pool, insertJob(t, pool, "fp-5"), "done", now)

		state, err := st.OpsState(context.Background())
		if err != nil {
			t.Fatalf("OpsState() error = %v", err)
		}
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

	t.Run("empty database reports zero", func(t *testing.T) {
		st, _ := newStore(t)
		state, err := st.OpsState(context.Background())
		if err != nil {
			t.Fatalf("OpsState() error = %v", err)
		}
		want := dto.OpsState{HarvestAge: map[string]time.Duration{}}
		if diff := cmp.Diff(want, state); diff != "" {
			t.Errorf("OpsState() mismatch (-want +got):\n%s", diff)
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

		state, err := st.OpsState(context.Background())
		if err != nil {
			t.Fatalf("OpsState() error = %v", err)
		}
		if state.BoardsOverdue != 2 {
			t.Errorf("OpsState().BoardsOverdue = %d, want %d", state.BoardsOverdue, 2)
		}
	})

	t.Run("counts verified boards with three consecutive failures as failing", func(t *testing.T) {
		st, pool := newStore(t)
		due := time.Now().Add(time.Hour)
		insertBoard(t, pool, "verified", due, nil, 3)
		insertBoard(t, pool, "verified", due, nil, 2)
		insertBoard(t, pool, "retired", due, nil, 5)

		state, err := st.OpsState(context.Background())
		if err != nil {
			t.Fatalf("OpsState() error = %v", err)
		}
		if state.BoardsFailing != 1 {
			t.Errorf("OpsState().BoardsFailing = %d, want %d", state.BoardsFailing, 1)
		}
	})

	t.Run("counts source targets whose last run failed", func(t *testing.T) {
		st, pool := newStore(t)
		userID := insertUser(t, pool)
		for i, status := range []string{"failed", "failed", "succeeded", "idle"} {
			if _, err := pool.Exec(context.Background(),
				`INSERT INTO source_targets (user_id, source, value, run_status) VALUES ($1, 'linkedin', $2, $3)`,
				userID, fmt.Sprintf("q-%d", i), status); err != nil {
				t.Fatalf("insert source_target: %v", err)
			}
		}

		state, err := st.OpsState(context.Background())
		if err != nil {
			t.Fatalf("OpsState() error = %v", err)
		}
		if state.SourceTargetsFailed != 2 {
			t.Errorf("OpsState().SourceTargetsFailed = %d, want %d", state.SourceTargetsFailed, 2)
		}
	})

	t.Run("reports harvest age per harvester", func(t *testing.T) {
		st, pool := newStore(t)
		now := time.Now()
		for harvester, at := range map[string]time.Time{"ashby": now.Add(-3 * time.Hour), "lever": now.Add(-time.Minute)} {
			if _, err := pool.Exec(context.Background(), `INSERT INTO harvest_runs (harvester, last_succeeded_at) VALUES ($1, $2)`, harvester, at); err != nil {
				t.Fatalf("insert harvest_runs: %v", err)
			}
		}

		state, err := st.OpsState(context.Background())
		if err != nil {
			t.Fatalf("OpsState() error = %v", err)
		}
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
}

func TestQueueMissingAnswers_QueuesOnlyOpenScoredFingerprintedJobsMissingHash(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()

	userA := insertUser(t, pool)
	userB := insertUser(t, pool)
	missing := insertJob(t, pool, "fp-missing")
	closed := insertJob(t, pool, "fp-closed")
	answered := insertJob(t, pool, "fp-answered")
	otherUsers := insertJob(t, pool, "fp-other-user")

	for _, jobID := range []string{missing, closed, answered} {
		if _, err := pool.Exec(ctx, "INSERT INTO job_scores (job_id, user_id) VALUES ($1, $2)", jobID, userA); err != nil {
			t.Fatalf("seed job_scores: %v", err)
		}
	}
	if _, err := pool.Exec(ctx, "INSERT INTO job_scores (job_id, user_id) VALUES ($1, $2)", otherUsers, userB); err != nil {
		t.Fatalf("seed job_scores: %v", err)
	}
	if _, err := pool.Exec(ctx, "UPDATE jobs SET closed_at = NOW() WHERE id = $1", closed); err != nil {
		t.Fatalf("close job: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO option_answers (job_id, fingerprint, question_hash, model, p_yes, p_no, p_not_stated, confidence)
		 VALUES ($1, 'fp-answered', $2, 'typesafe/jev-1.13', 0.9, 0.05, 0.05, 0.9)`,
		answered, "hash-1"); err != nil {
		t.Fatalf("seed option_answers: %v", err)
	}

	n, err := st.QueueMissingAnswers(ctx, userA, []string{"hash-1"}, "typesafe/jev-1.13")
	if err != nil {
		t.Fatalf("QueueMissingAnswers: %v", err)
	}
	if n != 1 {
		t.Fatalf("queued = %d, want 1", n)
	}

	for jobID, want := range map[string]int{missing: 1, closed: 0, answered: 0, otherUsers: 0} {
		var count int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM effect_outbox WHERE job_id = $1", jobID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Errorf("effects for job %s = %d, want %d", jobID, count, want)
		}
	}

	var firstDiscovery bool
	if err := pool.QueryRow(ctx, "SELECT first_discovery FROM effect_outbox WHERE job_id = $1", missing).Scan(&firstDiscovery); err != nil {
		t.Fatal(err)
	}
	if firstDiscovery {
		t.Error("first_discovery = true, want false (no alert on backfill)")
	}
}

func TestQueueMissingAnswers_QuiescesWhenNoGapOrEffectPending(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()

	userID := insertUser(t, pool)
	jobID := insertJob(t, pool, "fp-1")
	if _, err := pool.Exec(ctx, "INSERT INTO job_scores (job_id, user_id) VALUES ($1, $2)", jobID, userID); err != nil {
		t.Fatalf("seed job_scores: %v", err)
	}

	n, err := st.QueueMissingAnswers(ctx, userID, []string{"hash-1"}, "typesafe/jev-1.13")
	if err != nil {
		t.Fatalf("QueueMissingAnswers: %v", err)
	}
	if n != 1 {
		t.Fatalf("queued = %d, want 1", n)
	}

	n, err = st.QueueMissingAnswers(ctx, userID, []string{"hash-1"}, "typesafe/jev-1.13")
	if err != nil {
		t.Fatalf("QueueMissingAnswers: %v", err)
	}
	if n != 0 {
		t.Fatalf("queued while effect pending = %d, want 0", n)
	}

	if _, err := pool.Exec(ctx, "DELETE FROM effect_outbox WHERE job_id = $1", jobID); err != nil {
		t.Fatalf("clear outbox: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO option_answers (job_id, fingerprint, question_hash, model, p_yes, p_no, p_not_stated, confidence)
		 VALUES ($1, 'fp-1', $2, 'typesafe/jev-1.13', 0.9, 0.05, 0.05, 0.9)`,
		jobID, "hash-1"); err != nil {
		t.Fatalf("seed option_answers: %v", err)
	}

	n, err = st.QueueMissingAnswers(ctx, userID, []string{"hash-1"}, "typesafe/jev-1.13")
	if err != nil {
		t.Fatalf("QueueMissingAnswers: %v", err)
	}
	if n != 0 {
		t.Fatalf("queued with no gap = %d, want 0", n)
	}
}

func TestClaimAnswerEffect_ConcurrentClaimsExactlyOneWinner(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	jobID := insertJob(t, pool, "fp-1")
	insertEffect(t, pool, jobID, "fp-1")

	const claimers = 8
	var wins atomic.Int64
	var notFounds atomic.Int64
	var wg sync.WaitGroup
	wg.Add(claimers)
	for range claimers {
		go func() {
			defer wg.Done()
			_, err := st.ClaimAnswerEffect(ctx)
			switch {
			case err == nil:
				wins.Add(1)
			case errors.Is(err, data.ErrNotFound):
				notFounds.Add(1)
			default:
				t.Errorf("ClaimAnswerEffect: %v", err)
			}
		}()
	}
	wg.Wait()

	if wins.Load() != 1 {
		t.Fatalf("winners = %d, want exactly 1", wins.Load())
	}
	if notFounds.Load() != claimers-1 {
		t.Fatalf("losers = %d, want %d", notFounds.Load(), claimers-1)
	}
}

func TestSaveAnswers_KeepsExistingAndRoundTrips(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	jobID := insertJob(t, pool, "fp-1")

	if err := st.SaveAnswers(ctx, jobID, "fp-1", "m", map[string]dto.Answer{"h1": {PYes: 0.5}}); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveAnswers(ctx, jobID, "fp-1", "m", map[string]dto.Answer{"h1": {PYes: 0.9}, "h2": {PNo: 0.5}}); err != nil {
		t.Fatal(err)
	}
	got, err := st.ListAnswers(ctx, jobID, "fp-1", "m")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got["h1"].PYes != 0.5 || got["h2"].PNo != 0.5 {
		t.Fatalf("answers = %+v, want h1 kept at 0.5 and h2 added", got)
	}
}

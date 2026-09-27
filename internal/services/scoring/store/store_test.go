package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/store"
)

func newStore(t *testing.T) (*store.Store, *pgxpool.Pool) {
	t.Helper()
	pool := pgtest.New(t)
	return store.New(pool), pool
}

func insertUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (username, password_hash) VALUES ($1, 'hash') RETURNING id`,
		"user-"+t.Name()).Scan(&id)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return id
}

func insertJob(t *testing.T, pool *pgxpool.Pool, fingerprint string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO jobs (title, location, url, company_slug, source, updated_at, content_fingerprint)
		 VALUES ('Engineer', 'Remote', $1, 'acme', 'greenhouse', NOW(), $2) RETURNING id`,
		"https://example.com/"+t.Name(), fingerprint).Scan(&id)
	if err != nil {
		t.Fatalf("insert job: %v", err)
	}
	return id
}

func insertScoringOption(t *testing.T, pool *pgxpool.Pool, id, dimension, label, question string) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO scoring_options (id, dimension, label, question) VALUES ($1, $2, $3, $4)`,
		id, dimension, label, question)
	if err != nil {
		t.Fatalf("insert scoring option: %v", err)
	}
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

	inputs, err := st.ListScoringInputs(ctx, userID)
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

func TestOpsState_CountsPendingAndFailed(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	jobID := insertJob(t, pool, "fp-1")
	insertEffect(t, pool, jobID, "fp-1")

	state, err := st.OpsState(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if state.OutboxPending != 1 || state.OutboxFailed != 0 {
		t.Fatalf("state = %+v, want 1 pending, 0 failed", state)
	}
}

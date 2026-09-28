package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
)

func newStore(t *testing.T) (*store.Store, *pgxpool.Pool) {
	t.Helper()
	pool := pgtest.New(t)
	return store.New(pool, scoring.NewFacade(pool)), pool
}

func TestStoreSatisfiesContract(t *testing.T) {
	jobsearchtest.RunStoreContract(t, func(t *testing.T) (jobsearchtest.Store, string) {
		t.Helper()
		st, pool := newStore(t)
		return st, insertUser(t, pool)
	})
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

var baseJob = dto.Job{
	Title:       "Software Engineer",
	Location:    "London",
	URL:         "https://example.com/jobs/1",
	CompanySlug: "example",
	Source:      "greenhouse",
	UpdatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
}

func effectCountForJob(t *testing.T, pool *pgxpool.Pool, jobID string) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM effect_outbox WHERE job_id = $1", jobID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func TestPageJobsKeepsPositionUnderInsert(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	userID := insertUser(t, pool)
	company := "10000000-0000-0000-0000-000000000001"
	if _, err := pool.Exec(ctx, `INSERT INTO companies (id,slug,name) VALUES ($1,'page-jobs-test-acme','Acme')`, company); err != nil {
		t.Fatal(err)
	}
	insert := func(id string, day int, closed bool) {
		t.Helper()
		_, err := pool.Exec(ctx, `INSERT INTO jobs (id,title,location,url,company_slug,source,updated_at,scraped_at,description,company_id,closed_at) VALUES ($1::uuid,'Role','','https://example.com/'||$1::text,'acme','test',$2::timestamptz,$2::timestamptz,'full description',$3,CASE WHEN $4 THEN $2::timestamptz ELSE NULL END)`,
			id, time.Date(2026, 1, day, 0, 0, 0, 0, time.UTC), company, closed)
		if err != nil {
			t.Fatal(err)
		}
	}
	insert("20000000-0000-0000-0000-000000000001", 1, false)
	insert("20000000-0000-0000-0000-000000000002", 2, false)
	insert("20000000-0000-0000-0000-000000000003", 3, true)

	options := dto.JobPageOptions{Limit: 1, Availability: "open", CompanyID: company}
	first, err := st.Page(ctx, userID, options)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 1 || first.Items[0].ID != "20000000-0000-0000-0000-000000000002" || first.Items[0].Description != "" {
		t.Fatalf("first page: %+v", first)
	}
	detail, err := st.GetJob(ctx, first.Items[0].ID, userID)
	if err != nil || detail.Description != "full description" {
		t.Fatalf("detail: %+v, %v", detail, err)
	}

	insert("20000000-0000-0000-0000-000000000004", 4, false)
	options.CursorTime, options.CursorID = first.Items[0].ScrapedAt, first.Items[0].ID
	second, err := st.Page(ctx, userID, options)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Items[0].ID != "20000000-0000-0000-0000-000000000001" {
		t.Fatalf("second page: %+v", second)
	}

	closed, err := st.Page(ctx, userID, dto.JobPageOptions{Limit: 10, Availability: "closed", CompanyID: company})
	if err != nil || len(closed.Items) != 1 || closed.Items[0].ID != "20000000-0000-0000-0000-000000000003" {
		t.Fatalf("closed page: %+v, %v", closed, err)
	}
}

func TestListingsHideBlockedJob(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	userID := insertUser(t, pool)
	company := "10000000-0000-0000-0000-000000000002"
	if _, err := pool.Exec(ctx, `INSERT INTO companies (id,slug,name) VALUES ($1,'blocked-job-test-acme','Acme')`, company); err != nil {
		t.Fatal(err)
	}
	blockedID := "30000000-0000-0000-0000-000000000001"
	openID := "30000000-0000-0000-0000-000000000002"
	insert := func(id string) {
		t.Helper()
		_, err := pool.Exec(ctx, `INSERT INTO jobs (id,title,location,url,company_slug,source,updated_at,scraped_at,description,company_id) VALUES ($1::uuid,'Role','','https://example.com/'||$1::text,'acme','test',NOW(),NOW(),'full description',$2)`,
			id, company)
		if err != nil {
			t.Fatal(err)
		}
	}
	insert(blockedID)
	insert(openID)
	if _, err := pool.Exec(ctx,
		`INSERT INTO job_scores (job_id, user_id, suitability_score, breakdown) VALUES ($1::uuid, $2::uuid, 0, $3::jsonb)`,
		blockedID, userID, `[{"key":"domain:gambling","label":"Gambling","stance":"block","resolved":"yes","effect":"blocked"}]`,
	); err != nil {
		t.Fatal(err)
	}

	page, err := st.Page(ctx, userID, dto.JobPageOptions{Limit: 10, Availability: "open", CompanyID: company})
	if err != nil {
		t.Fatal(err)
	}
	for _, job := range page.Items {
		if job.ID == blockedID {
			t.Fatalf("Page returned blocked job %s", blockedID)
		}
	}
	if len(page.Items) != 1 || page.Items[0].ID != openID {
		t.Fatalf("Page items = %+v, want only %s", page.Items, openID)
	}

	all, err := st.ListJobs(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	for _, job := range all {
		if job.ID == blockedID {
			t.Fatalf("ListJobs returned blocked job %s", blockedID)
		}
	}
}

func TestSaveCanonicalAliasesAndReplayKeepsOneJob(t *testing.T) {
	st, _ := newStore(t)
	ctx := context.Background()
	first := baseJob
	first.BoardID = "11111111-1111-1111-1111-111111111111"
	first.ProviderPostingID = "posting-1"
	first.URL = "https://example.com/jobs/1?ref=board"

	saved, status, err := st.SaveCanonical(ctx, first)
	if err != nil || status != "new" || saved.ID == "" {
		t.Fatalf("first save: status=%q job=%+v err=%v", status, saved, err)
	}

	alias := first
	alias.URL = "https://example.com/jobs/1?ref=partner"
	savedAgain, status, err := st.SaveCanonical(ctx, alias)
	if err != nil || status != "unchanged" || savedAgain.ID != saved.ID {
		t.Fatalf("alias: status=%q job=%+v err=%v", status, savedAgain, err)
	}

	replayed, status, err := st.SaveCanonical(ctx, alias)
	if err != nil || status != "unchanged" || replayed.ID != saved.ID {
		t.Fatalf("replay: status=%q job=%+v err=%v", status, replayed, err)
	}

	jobs, err := st.ListJobs(ctx, "")
	if err != nil || len(jobs) != 1 {
		t.Fatalf("list: count=%d err=%v", len(jobs), err)
	}
	if jobs[0].BoardID != first.BoardID || jobs[0].ProviderPostingID != first.ProviderPostingID || jobs[0].ContentFingerprint == "" {
		t.Fatalf("canonical identity missing from read: %+v", jobs[0])
	}
}

func TestSaveCanonicalQueuesOneAnswerEffectPerContentVersion(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	userID := insertUser(t, pool)
	company, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "outbox-company", Name: "Outbox Company"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.SetCompanyTracking(ctx, userID, company.ID, true, 360); err != nil {
		t.Fatal(err)
	}
	job := baseJob
	job.URL = "https://example.com/jobs/outbox"
	job.CompanySlug = company.Slug
	job.CompanyID = company.ID

	var jobID string
	for _, title := range []string{"Engineer", "Engineer", "Senior Engineer"} {
		job.Title = title
		saved, _, err := st.SaveCanonical(ctx, job)
		if err != nil {
			t.Fatal(err)
		}
		jobID = saved.ID
	}
	if count := effectCountForJob(t, pool, jobID); count != 2 {
		t.Fatalf("queued answer effects = %d, want 2", count)
	}
}

func TestSaveCanonicalConflictingBoardCannotClaimURL(t *testing.T) {
	st, _ := newStore(t)
	ctx := context.Background()
	first := baseJob
	first.BoardID = "11111111-1111-1111-1111-111111111111"
	first.ProviderPostingID = "posting-1"
	if _, _, err := st.SaveCanonical(ctx, first); err != nil {
		t.Fatal(err)
	}

	conflict := baseJob
	conflict.BoardID = "22222222-2222-2222-2222-222222222222"
	conflict.ProviderPostingID = "posting-2"
	if _, _, err := st.SaveCanonical(ctx, conflict); !errors.Is(err, store.ErrCanonicalConflict) {
		t.Fatalf("err = %v, want ErrCanonicalConflict", err)
	}
}

func TestSetCompanyTrackingBackfillsFingerprintsAndQueuesScores(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	userID := insertUser(t, pool)
	company, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "backfill-co", Name: "Backfill Co"})
	if err != nil {
		t.Fatal(err)
	}
	job := baseJob
	job.URL = "https://example.com/jobs/backfill"
	job.CompanySlug = company.Slug
	saved, _, err := st.SaveCanonical(ctx, job)
	if err != nil {
		t.Fatal(err)
	}

	tracking, err := st.SetCompanyTracking(ctx, userID, company.ID, true, 120)
	if err != nil {
		t.Fatal(err)
	}
	if !tracking.Enabled || tracking.CheckIntervalMinutes != 120 {
		t.Fatalf("tracking = %+v", tracking)
	}
	if count := effectCountForJob(t, pool, saved.ID); count == 0 {
		t.Fatalf("queued answer effects = %d, want > 0", count)
	}
}

func TestCompanyBoardsRoundTrip(t *testing.T) {
	st, _ := newStore(t)
	ctx := context.Background()
	company, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "board-co", Name: "Board Co"})
	if err != nil {
		t.Fatal(err)
	}

	board, err := st.UpsertCandidateBoard(ctx, company.ID, "greenhouse", "board-co")
	if err != nil {
		t.Fatal(err)
	}
	if board.Status != dto.BoardCandidate {
		t.Fatalf("status = %q, want candidate", board.Status)
	}

	boards, err := st.ListCompanyBoards(ctx, company.ID)
	if err != nil || len(boards) != 1 || boards[0].ID != board.ID {
		t.Fatalf("boards = %+v, err = %v", boards, err)
	}

	if _, err := st.GetVerifiedBoardID(ctx, "greenhouse", "board-co"); !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("unverified board: err = %v, want ErrNotFound", err)
	}
}

func TestSourceTargetLifecycle(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	userID := insertUser(t, pool)

	target, err := st.CreateSourceTarget(ctx, userID, "linkedin", "acme", true, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateSourceTarget(ctx, userID, "linkedin", "acme", true, map[string]string{}); !errors.Is(err, store.ErrSourceTargetExists) {
		t.Fatalf("duplicate create: err = %v, want ErrSourceTargetExists", err)
	}

	disabled := false
	updated, err := st.UpdateSourceTarget(ctx, target.ID, userID, &disabled, nil)
	if err != nil || updated.Enabled {
		t.Fatalf("update: target = %+v, err = %v", updated, err)
	}

	started, err := st.StartSourceTargetRun(ctx, target.ID)
	if err != nil || started.RunStatus != "queued" {
		t.Fatalf("start run: target = %+v, err = %v", started, err)
	}

	if err := st.DeleteSourceTarget(ctx, target.ID, userID); err != nil {
		t.Fatal(err)
	}
	targets, err := st.ListSourceTargetsByUser(ctx, userID)
	if err != nil || len(targets) != 0 {
		t.Fatalf("targets after delete = %+v, err = %v", targets, err)
	}
}

func TestCandidateSaveListAndAssess(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	userID := insertUser(t, pool)
	target, err := st.CreateSourceTarget(ctx, userID, "linkedin", "candidate-search", true, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}

	saved, err := st.SaveCards(ctx, target, []dto.Job{{URL: "https://example.com/candidate/1", Title: "Engineer", CompanySlug: "acme"}})
	if err != nil || len(saved) != 1 {
		t.Fatalf("save cards: %+v, %v", saved, err)
	}

	listed, err := st.ListForUser(ctx, userID, "", 10)
	if err != nil || len(listed) != 1 || listed[0].ID != saved[0].ID {
		t.Fatalf("list for user: %+v, %v", listed, err)
	}

	queueDetail, err := st.Assess(ctx, saved[0].ID, userID, time.Now(), true)
	if err != nil || !queueDetail {
		t.Fatalf("assess: queueDetail=%v err=%v", queueDetail, err)
	}
	if err := st.MarkDetailPending(ctx, saved[0].ID); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteExpiredCandidates(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	userID := insertUser(t, pool)
	target, err := st.CreateSourceTarget(ctx, userID, "linkedin", "expiry-search", true, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	saved, err := st.SaveCards(ctx, target, []dto.Job{{URL: "https://example.com/candidate/expiring", Title: "Engineer", CompanySlug: "acme"}})
	if err != nil || len(saved) != 1 {
		t.Fatalf("save cards: %+v, %v", saved, err)
	}
	if _, err := pool.Exec(ctx, "UPDATE job_candidates SET expires_at = NOW() - INTERVAL '1 day' WHERE id = $1::uuid", saved[0].ID); err != nil {
		t.Fatal(err)
	}

	if err := st.DeleteExpiredCandidates(ctx); err != nil {
		t.Fatal(err)
	}

	listed, err := st.ListForUser(ctx, userID, "", 10)
	if err != nil || len(listed) != 0 {
		t.Fatalf("list after expiry = %+v, err = %v", listed, err)
	}
}

func TestNewURLs(t *testing.T) {
	st, _ := newStore(t)
	ctx := context.Background()
	job := baseJob
	job.URL = "https://example.com/jobs/new-urls"
	if _, _, err := st.SaveCanonical(ctx, job); err != nil {
		t.Fatal(err)
	}

	newURLs, err := st.NewURLs(ctx, []string{job.URL, "https://example.com/jobs/never-seen"})
	if err != nil {
		t.Fatal(err)
	}
	if len(newURLs) != 1 || newURLs[0] != "https://example.com/jobs/never-seen" {
		t.Fatalf("new URLs = %v", newURLs)
	}
}

func TestListCompaniesToCrawlAndTouch(t *testing.T) {
	st, _ := newStore(t)
	ctx := context.Background()
	company, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "crawl-co", Name: "Crawl Co", Domain: "crawl-co.example"})
	if err != nil {
		t.Fatal(err)
	}

	due, err := st.ListCompaniesToCrawl(ctx, 10)
	if err != nil || len(due) != 1 || due[0].ID != company.ID {
		t.Fatalf("due for crawl = %+v, err = %v", due, err)
	}

	if err := st.TouchCompanyCrawled(ctx, company.ID); err != nil {
		t.Fatal(err)
	}
	due, err = st.ListCompaniesToCrawl(ctx, 10)
	if err != nil || len(due) != 0 {
		t.Fatalf("after touch = %+v, err = %v", due, err)
	}
}

func TestSourceTargetRunRecovery(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	userID := insertUser(t, pool)
	target, err := st.CreateSourceTargetWithRun(ctx, userID, "linkedin", "recovery-search", true, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}

	got, err := st.GetSourceTarget(ctx, target.ID)
	if err != nil || got.ID != target.ID {
		t.Fatalf("get target = %+v, err = %v", got, err)
	}
	if _, err := st.GetSourceTarget(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("missing target: err = %v, want ErrNotFound", err)
	}

	running, err := st.TransitionSourceTargetRun(ctx, target.ID, target.RunID, "running", "")
	if err != nil || running.RunStatus != "running" {
		t.Fatalf("transition to running = %+v, err = %v", running, err)
	}

	if _, err := pool.Exec(ctx, "UPDATE source_targets SET updated_at = NOW() - INTERVAL '1 hour' WHERE id = $1::uuid", target.ID); err != nil {
		t.Fatal(err)
	}
	recoverable, err := st.ListRecoverableSourceTargets(ctx)
	if err != nil || len(recoverable) != 1 || recoverable[0].ID != target.ID {
		t.Fatalf("recoverable = %+v, err = %v", recoverable, err)
	}

	claimed, err := st.ClaimRecoverableSourceTarget(ctx, target.ID, target.RunID)
	if err != nil || claimed.ID != target.ID {
		t.Fatalf("claim recoverable = %+v, err = %v", claimed, err)
	}
	if _, err := st.ClaimRecoverableSourceTarget(ctx, target.ID, target.RunID); !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("re-claim before stale again: err = %v, want ErrNotFound", err)
	}

	succeeded, err := st.TransitionSourceTargetRun(ctx, target.ID, target.RunID, "succeeded", "")
	if err != nil || succeeded.RunStatus != "succeeded" {
		t.Fatalf("transition to succeeded = %+v, err = %v", succeeded, err)
	}
}

func boardFixture(t *testing.T, st *store.Store, pool *pgxpool.Pool) (context.Context, dto.CompanyBoard, string) {
	t.Helper()
	ctx := context.Background()
	userID := insertUser(t, pool)
	company, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "poll-co", Name: "Poll Co"})
	if err != nil {
		t.Fatal(err)
	}
	board, err := st.UpsertCandidateBoard(ctx, company.ID, "greenhouse", "poll-co")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.SetCompanyTracking(ctx, userID, company.ID, true, 60); err != nil {
		t.Fatal(err)
	}
	return ctx, board, company.ID
}

func TestBoardPollDueClaimAndEmptyClosure(t *testing.T) {
	st, pool := newStore(t)
	ctx, board, companyID := boardFixture(t, st, pool)

	if due, err := st.ListDueBoards(ctx); err != nil || len(due) != 0 {
		t.Fatalf("candidate due = %v, err = %v", due, err)
	}
	if _, err := st.VerifyCompanyBoard(ctx, companyID, board.Source, board.BoardToken, "user_confirmed"); err != nil {
		t.Fatal(err)
	}
	due, err := st.ListDueBoards(ctx)
	if err != nil || len(due) != 1 || due[0].ID != board.ID {
		t.Fatalf("verified due = %v, err = %v", due, err)
	}

	claim, err := st.ClaimBoard(ctx, board.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.ClaimBoard(ctx, board.ID, false); !errors.Is(err, store.ErrBoardClaimUnavailable) {
		t.Fatalf("second claim = %v, want ErrBoardClaimUnavailable", err)
	}

	job := dto.Job{Title: "Engineer", URL: "https://boards.greenhouse.io/poll-co/jobs/1", Source: "greenhouse", CompanySlug: "poll-co", CompanyID: companyID, BoardID: board.ID, UpdatedAt: time.Now()}
	if _, _, err := st.SaveCanonical(ctx, job); err != nil {
		t.Fatal(err)
	}
	if err := st.CompleteBoard(ctx, dto.BoardSnapshot{Poll: claim, Complete: true, Jobs: []dto.Job{job}}); err != nil {
		t.Fatal(err)
	}
	if err := st.CompleteBoard(ctx, dto.BoardSnapshot{Poll: claim, Complete: true, Jobs: []dto.Job{job}}); !errors.Is(err, store.ErrBoardClaimUnavailable) {
		t.Fatalf("replay = %v, want ErrBoardClaimUnavailable", err)
	}
	if due, err := st.ListDueBoards(ctx); err != nil || len(due) != 0 {
		t.Fatalf("completed due = %v, err = %v", due, err)
	}

	for n := 1; n <= 2; n++ {
		empty, err := st.ClaimBoard(ctx, board.ID, true)
		if err != nil {
			t.Fatal(err)
		}
		if err := st.CompleteBoard(ctx, dto.BoardSnapshot{Poll: empty, Complete: true}); err != nil {
			t.Fatal(err)
		}
		var closed bool
		if err := pool.QueryRow(ctx, "SELECT closed_at IS NOT NULL FROM jobs WHERE url = $1", job.URL).Scan(&closed); err != nil {
			t.Fatal(err)
		}
		if closed != (n == 2) {
			t.Fatalf("after %d empty snapshots closed = %t", n, closed)
		}
	}
}

func TestFailBoardRequiresActiveLease(t *testing.T) {
	st, pool := newStore(t)
	ctx, board, companyID := boardFixture(t, st, pool)
	if _, err := st.VerifyCompanyBoard(ctx, companyID, board.Source, board.BoardToken, "user_confirmed"); err != nil {
		t.Fatal(err)
	}
	claim, err := st.ClaimBoard(ctx, board.ID, false)
	if err != nil {
		t.Fatal(err)
	}

	if err := st.FailBoard(ctx, claim); err != nil {
		t.Fatal(err)
	}
	if err := st.FailBoard(ctx, claim); !errors.Is(err, store.ErrBoardClaimUnavailable) {
		t.Fatalf("stale fail = %v, want ErrBoardClaimUnavailable", err)
	}
}

func insertNamedUser(t *testing.T, pool *pgxpool.Pool, username string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO users (username, password_hash) VALUES ($1, 'hash') RETURNING id`, username).Scan(&id); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return id
}

func insertOptionAnswer(t *testing.T, pool *pgxpool.Pool, jobID, fingerprint, questionHash string) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		"INSERT INTO option_answers (job_id, fingerprint, question_hash, model, p_yes, p_no, p_not_stated, confidence) VALUES ($1, $2, $3, 'test-model', 0.5, 0.3, 0.2, 0.9)",
		jobID, fingerprint, questionHash)
	if err != nil {
		t.Fatalf("insert option_answer: %v", err)
	}
}

func optionAnswerFingerprints(t *testing.T, pool *pgxpool.Pool, jobID string) map[string]int {
	t.Helper()
	rows, err := pool.Query(context.Background(), "SELECT fingerprint FROM option_answers WHERE job_id = $1", jobID)
	if err != nil {
		t.Fatalf("query option_answers: %v", err)
	}
	defer rows.Close()
	counts := make(map[string]int)
	for rows.Next() {
		var fp string
		if err := rows.Scan(&fp); err != nil {
			t.Fatalf("scan fingerprint: %v", err)
		}
		counts[fp]++
	}
	return counts
}

func TestUpsertCompanyConflictMerge(t *testing.T) {
	t.Run("insert", func(t *testing.T) {
		st, _ := newStore(t)
		ctx := context.Background()
		c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme", ATSSource: "greenhouse", ATSToken: "acme"})
		if err != nil {
			t.Fatalf("UpsertCompany: %v", err)
		}
		if c.Slug != "acme" || c.Name != "Acme" || c.ATSSource != "greenhouse" || c.ATSToken != "acme" {
			t.Errorf("unexpected company: %+v", c)
		}
	})

	t.Run("conflict fills in missing ats fields", func(t *testing.T) {
		st, _ := newStore(t)
		ctx := context.Background()
		first, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme"})
		if err != nil {
			t.Fatalf("first UpsertCompany: %v", err)
		}
		if first.ATSSource != "" {
			t.Fatalf("expected empty ats source on discovery-first insert, got %q", first.ATSSource)
		}
		second, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme Corp", ATSSource: "greenhouse", ATSToken: "acme"})
		if err != nil {
			t.Fatalf("second UpsertCompany: %v", err)
		}
		if second.ID != first.ID {
			t.Errorf("expected same row, got different IDs")
		}
		if second.ATSSource != "greenhouse" || second.ATSToken != "acme" {
			t.Errorf("expected ats fields filled in, got source=%q token=%q", second.ATSSource, second.ATSToken)
		}
	})

	t.Run("conflict never overwrites an existing ats board", func(t *testing.T) {
		st, _ := newStore(t)
		ctx := context.Background()
		if _, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme", ATSSource: "greenhouse", ATSToken: "acme"}); err != nil {
			t.Fatalf("first UpsertCompany: %v", err)
		}
		got, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme", ATSSource: "lever", ATSToken: "acme-other"})
		if err != nil {
			t.Fatalf("second UpsertCompany: %v", err)
		}
		if got.ATSSource != "greenhouse" || got.ATSToken != "acme" {
			t.Errorf("expected original ats board preserved, got source=%q token=%q", got.ATSSource, got.ATSToken)
		}
	})

	t.Run("conflict fills in missing domain and linkedin id", func(t *testing.T) {
		st, _ := newStore(t)
		ctx := context.Background()
		first, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme"})
		if err != nil {
			t.Fatalf("first UpsertCompany: %v", err)
		}
		second, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme", Domain: "acme.com", LinkedInCompanyID: "12345"})
		if err != nil {
			t.Fatalf("second UpsertCompany: %v", err)
		}
		if second.ID != first.ID {
			t.Errorf("expected same row, got different IDs")
		}
		if second.Domain != "acme.com" || second.LinkedInCompanyID != "12345" {
			t.Errorf("expected domain/linkedin id filled in, got domain=%q linkedin=%q", second.Domain, second.LinkedInCompanyID)
		}
	})

	t.Run("conflict never overwrites an existing domain or linkedin id", func(t *testing.T) {
		st, _ := newStore(t)
		ctx := context.Background()
		if _, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme", Domain: "acme.com", LinkedInCompanyID: "12345"}); err != nil {
			t.Fatalf("first UpsertCompany: %v", err)
		}
		got, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme", Domain: "other.com", LinkedInCompanyID: "99999"})
		if err != nil {
			t.Fatalf("second UpsertCompany: %v", err)
		}
		if got.Domain != "acme.com" || got.LinkedInCompanyID != "12345" {
			t.Errorf("expected original domain/linkedin id preserved, got domain=%q linkedin=%q", got.Domain, got.LinkedInCompanyID)
		}
	})
}

func TestListCompaniesForUser(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	userID := insertUser(t, pool)

	tracked, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme", ATSSource: "greenhouse", ATSToken: "acme"})
	if err != nil {
		t.Fatalf("UpsertCompany tracked: %v", err)
	}
	if _, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "widgetco", Name: "Widgetco", ATSSource: "ashby", ATSToken: "widgetco"}); err != nil {
		t.Fatalf("UpsertCompany untracked: %v", err)
	}
	if _, err := st.SetCompanyTracking(ctx, userID, tracked.ID, true, 180); err != nil {
		t.Fatalf("SetCompanyTracking: %v", err)
	}

	companies, err := st.ListCompaniesForUser(ctx, userID)
	if err != nil {
		t.Fatalf("ListCompaniesForUser: %v", err)
	}
	if len(companies) != 2 {
		t.Fatalf("want 2 companies, got %d", len(companies))
	}
	for _, c := range companies {
		if c.ID == tracked.ID {
			if !c.Tracked || c.CheckIntervalMinutes != 180 {
				t.Errorf("expected acme tracking with 180-minute interval, got %+v", c)
			}
			continue
		}
		if c.Tracked {
			t.Errorf("expected widgetco to show as untracked")
		}
	}
}

func TestCompanyTrackingWithoutBoard(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	alice := insertNamedUser(t, pool, "tracking-alice")
	bob := insertNamedUser(t, pool, "tracking-bob")
	company, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "boardless", Name: "Boardless"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.SetCompanyTracking(ctx, alice, company.ID, true, 180); err != nil {
		t.Fatal(err)
	}
	if _, err := st.SetCompanyTracking(ctx, alice, company.ID, false, 0); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		userID       string
		wantTracked  bool
		wantInterval int
	}{
		{alice, false, 180},
		{bob, false, 0},
	} {
		companies, err := st.ListCompaniesForUser(ctx, test.userID)
		if err != nil {
			t.Fatal(err)
		}
		if len(companies) != 1 || companies[0].Tracked != test.wantTracked || companies[0].CheckIntervalMinutes != test.wantInterval {
			t.Errorf("user %s: got %+v", test.userID, companies)
		}
	}
}

func TestCompanyBoardConflict(t *testing.T) {
	st, _ := newStore(t)
	ctx := context.Background()
	first, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "other", Name: "Other"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpsertCandidateBoard(ctx, first.ID, "greenhouse", "acme"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpsertCandidateBoard(ctx, second.ID, "greenhouse", "acme"); !errors.Is(err, store.ErrBoardConflict) {
		t.Fatalf("want ErrBoardConflict, got %v", err)
	}
	if _, err := st.UpsertCandidateBoard(ctx, first.ID, "ashby", "acme"); err != nil {
		t.Fatal(err)
	}
	boards, err := st.ListCompanyBoards(ctx, first.ID)
	if err != nil || len(boards) != 2 {
		t.Fatalf("want two boards, got %+v, err = %v", boards, err)
	}
}

func TestCandidateRetentionDuplicateCardsAndAssessmentFlow(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	userID := insertUser(t, pool)
	target, err := st.CreateSourceTarget(ctx, userID, "wis", "engineer", true, nil)
	if err != nil {
		t.Fatal(err)
	}

	card := dto.Job{URL: "https://example.com/jobs/1#details", Title: "Senior Engineer", CompanySlug: "acme", Location: "London"}
	got, err := st.SaveCards(ctx, target, []dto.Job{card, card})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != got[1].ID || got[0].URL != "https://example.com/jobs/1" {
		t.Fatalf("duplicate candidates: %+v", got)
	}

	if _, err := pool.Exec(ctx, "UPDATE job_candidates SET expires_at = NOW() - INTERVAL '1 second'"); err != nil {
		t.Fatal(err)
	}
	retained, err := st.ListForUser(ctx, userID, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(retained) != 0 {
		t.Fatalf("expired candidate remained queryable: %+v", retained)
	}

	if _, err := st.SaveCards(ctx, target, []dto.Job{card}); err != nil {
		t.Fatal(err)
	}
	retained, err = st.ListForUser(ctx, userID, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(retained) != 1 {
		t.Fatalf("rediscovered candidate missing: %+v", retained)
	}
	if retained[0].Card.Source != target.Source {
		t.Fatalf("candidate source=%q, want %q", retained[0].Card.Source, target.Source)
	}

	version := time.Now().UTC()
	requested, err := st.Assess(ctx, got[0].ID, userID, version, false)
	if err != nil || requested {
		t.Fatalf("rejected assessment requested detail: requested=%v err=%v", requested, err)
	}
	requested, err = st.Assess(ctx, got[0].ID, userID, version.Add(time.Second), true)
	if err != nil || !requested {
		t.Fatalf("newly relevant candidate not requested: requested=%v err=%v", requested, err)
	}
	if err := st.MarkDetailPending(ctx, got[0].ID); err != nil {
		t.Fatal(err)
	}
	requested, err = st.Assess(ctx, got[0].ID, userID, version.Add(2*time.Second), true)
	if err != nil || requested {
		t.Fatalf("duplicate detail request: requested=%v err=%v", requested, err)
	}
	var assessments int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM candidate_assessments WHERE candidate_id = $1 AND user_id = $2", got[0].ID, userID).Scan(&assessments); err != nil {
		t.Fatal(err)
	}
	if assessments != 1 {
		t.Fatalf("assessment rows = %d, want one latest row", assessments)
	}
}

func TestBoardPollOmissionReopenAndRejectedCompletion(t *testing.T) {
	st, pool := newStore(t)
	ctx, board, companyID := boardFixture(t, st, pool)
	if _, err := st.VerifyCompanyBoard(ctx, companyID, board.Source, board.BoardToken, "user_confirmed"); err != nil {
		t.Fatal(err)
	}
	jobs := []dto.Job{
		{Title: "Engineer", URL: "https://boards.greenhouse.io/poll-co/jobs/1", Source: "greenhouse", CompanySlug: "poll-co", CompanyID: companyID, BoardID: board.ID, UpdatedAt: time.Now()},
		{Title: "Designer", URL: "https://boards.greenhouse.io/poll-co/jobs/2", Source: "greenhouse", CompanySlug: "poll-co", CompanyID: companyID, BoardID: board.ID, UpdatedAt: time.Now()},
	}
	for _, job := range jobs {
		if _, _, err := st.SaveCanonical(ctx, job); err != nil {
			t.Fatal(err)
		}
	}
	claim, err := st.ClaimBoard(ctx, board.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CompleteBoard(ctx, dto.BoardSnapshot{Poll: claim, Complete: false, Jobs: jobs}); err == nil {
		t.Fatal("partial snapshot accepted")
	}
	if err := st.CompleteBoard(ctx, dto.BoardSnapshot{Poll: claim, Complete: true, Jobs: append(jobs, dto.Job{URL: "https://boards.greenhouse.io/poll-co/jobs/missing"})}); err == nil {
		t.Fatal("missing ingest accepted")
	}
	var completed bool
	if err := pool.QueryRow(ctx, "SELECT last_completed_at IS NOT NULL FROM board_poll_state WHERE board_id = $1", board.ID).Scan(&completed); err != nil || completed {
		t.Fatalf("failed snapshot advanced freshness: %t, %v", completed, err)
	}
	if err := st.CompleteBoard(ctx, dto.BoardSnapshot{Poll: claim, Complete: true, Jobs: jobs}); err != nil {
		t.Fatal(err)
	}
	second, err := st.ClaimBoard(ctx, board.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CompleteBoard(ctx, dto.BoardSnapshot{Poll: second, Complete: true, Jobs: jobs[:1]}); err != nil {
		t.Fatal(err)
	}
	var closed bool
	if err := pool.QueryRow(ctx, "SELECT closed_at IS NOT NULL FROM jobs WHERE url = $1", jobs[1].URL).Scan(&closed); err != nil || !closed {
		t.Fatalf("omitted job closed=%t err=%v", closed, err)
	}
	third, err := st.ClaimBoard(ctx, board.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CompleteBoard(ctx, dto.BoardSnapshot{Poll: third, Complete: true, Jobs: jobs}); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT closed_at IS NOT NULL FROM jobs WHERE url = $1", jobs[1].URL).Scan(&closed); err != nil || closed {
		t.Fatalf("reappeared job closed=%t err=%v", closed, err)
	}
	if due, err := st.ListDueBoards(ctx); err != nil || len(due) != 0 {
		t.Fatalf("manual checks shifted cadence: %v %v", due, err)
	}
}

func TestBoardPollStaleClaimCannotOverwriteNewerCheck(t *testing.T) {
	st, pool := newStore(t)
	ctx, board, companyID := boardFixture(t, st, pool)
	if _, err := st.VerifyCompanyBoard(ctx, companyID, board.Source, board.BoardToken, "user_confirmed"); err != nil {
		t.Fatal(err)
	}
	old, err := st.ClaimBoard(ctx, board.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE board_poll_state SET lease_until = NOW() - INTERVAL '1 minute' WHERE board_id = $1", board.ID); err != nil {
		t.Fatal(err)
	}
	newer, err := st.ClaimBoard(ctx, board.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CompleteBoard(ctx, dto.BoardSnapshot{Poll: newer, Complete: true}); err != nil {
		t.Fatal(err)
	}
	if err := st.CompleteBoard(ctx, dto.BoardSnapshot{Poll: old, Complete: true}); !errors.Is(err, store.ErrBoardClaimUnavailable) {
		t.Fatalf("stale completion=%v", err)
	}
}

func TestBoardPollRetiresSupersededBoardAfterTwoEmptyChecks(t *testing.T) {
	st, pool := newStore(t)
	ctx, board, companyID := boardFixture(t, st, pool)
	if _, err := st.VerifyCompanyBoard(ctx, companyID, board.Source, board.BoardToken, "user_confirmed"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE company_boards SET superseded_at = NOW() WHERE id = $1", board.ID); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		claim, err := st.ClaimBoard(ctx, board.ID, true)
		if err != nil {
			t.Fatal(err)
		}
		if err := st.CompleteBoard(ctx, dto.BoardSnapshot{Poll: claim, Complete: true}); err != nil {
			t.Fatal(err)
		}
	}
	boards, err := st.ListCompanyBoards(ctx, companyID)
	if err != nil || len(boards) != 1 || boards[0].Status != dto.BoardRetired {
		t.Fatalf("board retirement=%v err=%v", boards, err)
	}
	if due, err := st.ListActiveBoards(ctx); err != nil || len(due) != 0 {
		t.Fatalf("retired board active=%v err=%v", due, err)
	}
}

func TestSaveCanonicalContentChangeAndDistinctBoard(t *testing.T) {
	st, _ := newStore(t)
	ctx := context.Background()
	first := baseJob
	first.BoardID = "11111111-1111-1111-1111-111111111111"
	first.ProviderPostingID = "posting-1"
	first.URL = "https://example.com/jobs/change/1"
	saved, _, err := st.SaveCanonical(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	changed := first
	changed.Title = "Senior Engineer"
	updated, status, err := st.SaveCanonical(ctx, changed)
	if err != nil || status != "changed" || updated.ID != saved.ID {
		t.Fatalf("changed: status=%q job=%+v err=%v", status, updated, err)
	}
	unchanged, status, err := st.SaveCanonical(ctx, changed)
	if err != nil || status != "unchanged" || unchanged.ID != saved.ID || unchanged.ContentFingerprint != updated.ContentFingerprint {
		t.Fatalf("changed replay: status=%q job=%+v err=%v", status, unchanged, err)
	}
	other := first
	other.BoardID = "22222222-2222-2222-2222-222222222222"
	other.URL = "https://other.example.com/jobs/change/1"
	distinct, status, err := st.SaveCanonical(ctx, other)
	if err != nil || status != "new" || distinct.ID == saved.ID {
		t.Fatalf("distinct board: status=%q job=%+v err=%v", status, distinct, err)
	}
}

func TestSaveCanonicalPrunesStaleOptionAnswersOnFingerprintChange(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()

	jobA := baseJob
	jobA.URL = "https://example.com/jobs/prune"
	saved, _, err := st.SaveCanonical(ctx, jobA)
	if err != nil {
		t.Fatal(err)
	}
	fpA := saved.ContentFingerprint

	jobB := jobA
	jobB.Title = "Staff Engineer"
	updatedB, status, err := st.SaveCanonical(ctx, jobB)
	if err != nil || status != "changed" {
		t.Fatalf("changed to B: status=%q err=%v", status, err)
	}
	fpB := updatedB.ContentFingerprint

	insertOptionAnswer(t, pool, saved.ID, fpA, "q-fpA")
	insertOptionAnswer(t, pool, saved.ID, fpB, "q-fpB")

	updatedA, status, err := st.SaveCanonical(ctx, jobA)
	if err != nil || status != "changed" || updatedA.ContentFingerprint != fpA {
		t.Fatalf("reverted to A: status=%q fingerprint=%q err=%v", status, updatedA.ContentFingerprint, err)
	}

	userID := insertUser(t, pool)
	if _, err := pool.Exec(ctx, "INSERT INTO job_scores (job_id, user_id) VALUES ($1, $2)", saved.ID, userID); err != nil {
		t.Fatal(err)
	}

	remaining := optionAnswerFingerprints(t, pool, saved.ID)
	if len(remaining) != 1 || remaining[fpA] != 1 {
		t.Fatalf("option_answers after prune = %v, want only {%q: 1}", remaining, fpA)
	}

	var scoredAnswerCount int
	err = pool.QueryRow(ctx,
		`SELECT count(*) FROM job_scores s
		 JOIN jobs j ON j.id = s.job_id
		 JOIN option_answers a ON a.job_id = j.id AND a.fingerprint = j.content_fingerprint
		 WHERE s.user_id = $1 AND a.question_hash = 'q-fpB'`, userID).Scan(&scoredAnswerCount)
	if err != nil {
		t.Fatalf("count scoring-visible answers: %v", err)
	}
	if scoredAnswerCount != 0 {
		t.Fatalf("pruned answer q-fpB still visible to scoring (matches the job's current fingerprint), want treated as unknown")
	}
}

func TestSaveCanonicalNoInterestedUserQueuesNoEffect(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	company, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "untracked-co", Name: "Untracked Co"})
	if err != nil {
		t.Fatal(err)
	}
	job := baseJob
	job.URL = "https://example.com/jobs/untracked"
	job.CompanySlug = company.Slug
	job.CompanyID = company.ID
	saved, _, err := st.SaveCanonical(ctx, job)
	if err != nil {
		t.Fatal(err)
	}
	if count := effectCountForJob(t, pool, saved.ID); count != 0 {
		t.Fatalf("queued answer effects for an untracked company = %d, want 0", count)
	}
}

func TestSaveCanonicalQueuesRegardlessOfExclusionFilters(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	company, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "filtered-co", Name: "Filtered Co"})
	if err != nil {
		t.Fatal(err)
	}
	userID := insertUser(t, pool)
	if _, err := pool.Exec(ctx,
		"INSERT INTO search_config (user_id, excluded_companies) VALUES ($1, $2)",
		userID, []string{"filtered-co"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.SetCompanyTracking(ctx, userID, company.ID, true, 360); err != nil {
		t.Fatal(err)
	}
	job := baseJob
	job.URL = "https://example.com/jobs/filtered"
	job.CompanySlug = company.Slug
	job.CompanyID = company.ID
	saved, _, err := st.SaveCanonical(ctx, job)
	if err != nil {
		t.Fatal(err)
	}
	if count := effectCountForJob(t, pool, saved.ID); count != 1 {
		t.Fatalf("queued answer effects = %d, want 1 (filters apply later, not at ingest)", count)
	}
}

func TestSourceTargetRunGenerationFencesStaleCompletion(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	userID := insertUser(t, pool)
	target, err := st.CreateSourceTarget(ctx, userID, "wis", "engineer", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	first, err := st.StartSourceTargetRun(ctx, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Enabled {
		t.Fatal("manual rerun did not enable target")
	}
	second, err := st.StartSourceTargetRun(ctx, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if first.RunID == second.RunID || first.RunID == "" {
		t.Fatalf("run IDs = %q, %q", first.RunID, second.RunID)
	}
	if _, err := st.TransitionSourceTargetRun(ctx, target.ID, first.RunID, "succeeded", ""); !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("stale completion error = %v", err)
	}
	current, err := st.TransitionSourceTargetRun(ctx, target.ID, second.RunID, "succeeded", "")
	if err != nil || current.RunStatus != "succeeded" {
		t.Fatalf("current completion = %+v, %v", current, err)
	}
}

func TestUpsertSourceTargetForCompanyToggle(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	userID := insertUser(t, pool)
	company, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme", ATSSource: "greenhouse", ATSToken: "acme"})
	if err != nil {
		t.Fatalf("UpsertCompany: %v", err)
	}

	created, err := st.UpsertSourceTargetForCompany(ctx, userID, "greenhouse", "acme", company.ID, true, 180)
	if err != nil {
		t.Fatalf("UpsertSourceTargetForCompany create: %v", err)
	}
	if !created.Enabled || created.CompanyID != company.ID || created.CheckIntervalMinutes != 180 {
		t.Fatalf("unexpected created target: %+v", created)
	}

	disabled, err := st.UpsertSourceTargetForCompany(ctx, userID, "greenhouse", "acme", company.ID, false, 0)
	if err != nil {
		t.Fatalf("UpsertSourceTargetForCompany disable: %v", err)
	}
	if disabled.ID != created.ID {
		t.Errorf("expected same target row on toggle, got different ID")
	}
	if disabled.Enabled {
		t.Errorf("expected target to be disabled")
	}
	if disabled.CheckIntervalMinutes != 180 {
		t.Errorf("expected interval preserved, got %d", disabled.CheckIntervalMinutes)
	}
}

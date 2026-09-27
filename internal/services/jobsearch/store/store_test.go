package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
)

func newStore(t *testing.T) (*store.Store, *pgxpool.Pool) {
	t.Helper()
	pool := pgtest.New(t)
	return store.New(pool, scoring.NewFacade(pool)), pool
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

func TestGetJobNotFound(t *testing.T) {
	st, pool := newStore(t)
	ctx := context.Background()
	userID := insertUser(t, pool)

	if _, err := st.GetJob(ctx, "00000000-0000-0000-0000-000000000000", userID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
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

	if _, err := st.GetVerifiedBoardID(ctx, "greenhouse", "board-co"); !errors.Is(err, store.ErrNotFound) {
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
	if _, err := st.GetSourceTarget(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, store.ErrNotFound) {
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
	if _, err := st.ClaimRecoverableSourceTarget(ctx, target.ID, target.RunID); !errors.Is(err, store.ErrNotFound) {
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

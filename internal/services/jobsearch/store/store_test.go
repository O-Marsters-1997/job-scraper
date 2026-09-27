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

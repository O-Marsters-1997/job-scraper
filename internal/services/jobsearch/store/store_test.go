package store_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/pgtest"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/services/sourcetargets"
)

var baseJob = dto.Job{
	Title:       "Software Engineer",
	Location:    "London",
	URL:         "https://example.com/jobs/1",
	CompanySlug: "example",
	Source:      "greenhouse",
	UpdatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
}

func newStore(t *testing.T) (*store.Store, *pgxpool.Pool) {
	t.Helper()
	pool := pgtest.New(t)
	return store.New(pool, scoring.NewFacade(pool)), pool
}

func newUserStore(t *testing.T) (*store.Store, *pgxpool.Pool, string) {
	t.Helper()
	st, pool := newStore(t)
	return st, pool, pgtest.InsertUser(t, pool)
}

func TestStoreSatisfiesContract(t *testing.T) {
	jobsearchtest.RunStoreContract(t, func(t *testing.T) (jobsearchtest.Store, string) {
		t.Helper()
		st, _, userID := newUserStore(t)
		return st, userID
	})
}

func insertCompany(t *testing.T, pool *pgxpool.Pool, slug string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(t.Context(), `INSERT INTO companies (slug,name) VALUES ($1,$1) RETURNING id`, slug).Scan(&id); err != nil {
		t.Fatalf("insert company %q: %v", slug, err)
	}
	return id
}

func insertJob(t *testing.T, pool *pgxpool.Pool, companyID string, n int, scrapedAt time.Time, closed bool) string {
	t.Helper()
	id := fmt.Sprintf("20000000-0000-0000-0000-%012d", n)
	_, err := pool.Exec(t.Context(),
		`INSERT INTO jobs (id,title,location,url,company_slug,source,updated_at,scraped_at,description,company_id,closed_at)
		 VALUES ($1::uuid,'Role','','https://example.com/'||$1::text,'acme','test',$2::timestamptz,$2::timestamptz,'full description',$3,CASE WHEN $4 THEN $2::timestamptz END)`,
		id, scrapedAt, companyID, closed)
	if err != nil {
		t.Fatalf("insert job %d: %v", n, err)
	}
	return id
}

func scoreJob(t *testing.T, pool *pgxpool.Pool, jobID, userID, breakdown string) {
	t.Helper()
	_, err := pool.Exec(t.Context(),
		`INSERT INTO job_scores (job_id, user_id, suitability_score, breakdown) VALUES ($1::uuid,$2::uuid,50,$3::jsonb)`,
		jobID, userID, breakdown)
	if err != nil {
		t.Fatalf("score job %s: %v", jobID, err)
	}
}

func upsertCompany(t *testing.T, st *store.Store, in dto.CompanyUpsert) dto.Company {
	t.Helper()
	company, err := st.UpsertCompany(t.Context(), in)
	if err != nil {
		t.Fatalf("UpsertCompany(%+v) err = %v", in, err)
	}
	return company
}

func saveJob(t *testing.T, st *store.Store, job dto.Job) (dto.Job, string) {
	t.Helper()
	saved, status, err := st.SaveCanonical(t.Context(), job)
	if err != nil {
		t.Fatalf("SaveCanonical(%s) err = %v", job.URL, err)
	}
	return saved, status
}

func trackCompany(t *testing.T, st *store.Store, userID, companyID string, minutes int) {
	t.Helper()
	if _, err := st.SetCompanyTracking(t.Context(), userID, companyID, true, minutes); err != nil {
		t.Fatalf("SetCompanyTracking(%s) err = %v", companyID, err)
	}
}

func createTarget(t *testing.T, st *store.Store, userID, source, value string) dto.SourceTarget {
	t.Helper()
	target, err := st.CreateSourceTarget(t.Context(), userID, source, value, true, nil)
	if err != nil {
		t.Fatalf("CreateSourceTarget(%s, %s) err = %v", source, value, err)
	}
	return target
}

func saveCards(t *testing.T, st *store.Store, target dto.SourceTarget, cards ...dto.Job) []sourcetargets.Candidate {
	t.Helper()
	saved, err := st.SaveCards(t.Context(), target, cards)
	if err != nil {
		t.Fatalf("SaveCards err = %v", err)
	}
	return saved
}

func effectCountForJob(t *testing.T, pool *pgxpool.Pool, jobID string) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(t.Context(), "SELECT count(*) FROM effect_outbox WHERE job_id = $1", jobID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func jobClosed(t *testing.T, pool *pgxpool.Pool, url string) bool {
	t.Helper()
	var closed bool
	if err := pool.QueryRow(t.Context(), "SELECT closed_at IS NOT NULL FROM jobs WHERE url = $1", url).Scan(&closed); err != nil {
		t.Fatal(err)
	}
	return closed
}

func jobIDs(jobs []dto.Job) []string {
	ids := make([]string, len(jobs))
	for i, job := range jobs {
		ids[i] = job.ID
	}
	return ids
}

func TestPage(t *testing.T) {
	t.Run("keeps position when a newer job is inserted", func(t *testing.T) {
		st, pool, userID := newUserStore(t)
		ctx := t.Context()
		company := insertCompany(t, pool, "acme")
		day := func(d int) time.Time { return time.Date(2026, 1, d, 0, 0, 0, 0, time.UTC) }
		oldest := insertJob(t, pool, company, 1, day(1), false)
		middle := insertJob(t, pool, company, 2, day(2), false)
		closed := insertJob(t, pool, company, 3, day(3), true)

		options := dto.JobPageOptions{Limit: 1, Availability: "open", CompanyID: company}
		first, err := st.Page(ctx, userID, options)
		if err != nil {
			t.Fatalf("Page(first) err = %v", err)
		}
		if diff := cmp.Diff([]string{middle}, jobIDs(first.Items)); diff != "" {
			t.Fatalf("first page ids (-want +got):\n%s", diff)
		}
		if first.Items[0].Description != "" {
			t.Errorf("listing carried description %q, want it left to GetJob", first.Items[0].Description)
		}
		detail, err := st.GetJob(ctx, middle, userID)
		if err != nil {
			t.Fatalf("GetJob(%s) err = %v", middle, err)
		}
		if detail.Description != "full description" {
			t.Errorf("GetJob(%s).Description = %q, want the full description", middle, detail.Description)
		}

		insertJob(t, pool, company, 4, day(4), false)
		options.CursorTime, options.CursorID = first.Items[0].ScrapedAt, first.Items[0].ID
		second, err := st.Page(ctx, userID, options)
		if err != nil {
			t.Fatalf("Page(second) err = %v", err)
		}
		if diff := cmp.Diff([]string{oldest}, jobIDs(second.Items)); diff != "" {
			t.Errorf("second page ids (-want +got):\n%s", diff)
		}

		closedPage, err := st.Page(ctx, userID, dto.JobPageOptions{Limit: 10, Availability: "closed", CompanyID: company})
		if err != nil {
			t.Fatalf("Page(closed) err = %v", err)
		}
		if diff := cmp.Diff([]string{closed}, jobIDs(closedPage.Items)); diff != "" {
			t.Errorf("closed page ids (-want +got):\n%s", diff)
		}
	})

	t.Run("hides a blocked job from listings", func(t *testing.T) {
		st, pool, userID := newUserStore(t)
		ctx := t.Context()
		company := insertCompany(t, pool, "acme")
		blocked := insertJob(t, pool, company, 1, time.Now(), false)
		open := insertJob(t, pool, company, 2, time.Now(), false)
		scoreJob(t, pool, blocked, userID, `[{"key":"domain:gambling","label":"Gambling","stance":"block","resolved":"yes","effect":"blocked"}]`)

		page, err := st.Page(ctx, userID, dto.JobPageOptions{Limit: 10, Availability: "open", CompanyID: company})
		if err != nil {
			t.Fatalf("Page() err = %v", err)
		}
		if diff := cmp.Diff([]string{open}, jobIDs(page.Items)); diff != "" {
			t.Errorf("Page() ids (-want +got):\n%s", diff)
		}
		all, err := st.ListJobs(ctx, userID)
		if err != nil {
			t.Fatalf("ListJobs() err = %v", err)
		}
		if diff := cmp.Diff([]string{open}, jobIDs(all)); diff != "" {
			t.Errorf("ListJobs() ids (-want +got):\n%s", diff)
		}
	})

	t.Run("scored only keeps the caller's scored jobs of the company", func(t *testing.T) {
		st, pool, userID := newUserStore(t)
		otherUserID := pgtest.InsertUser(t, pool)
		acme := insertCompany(t, pool, "acme")
		other := insertCompany(t, pool, "other")
		scored := insertJob(t, pool, acme, 1, time.Now(), false)
		insertJob(t, pool, acme, 2, time.Now(), false)
		scoredByOther := insertJob(t, pool, acme, 3, time.Now(), false)
		otherCompany := insertJob(t, pool, other, 4, time.Now(), false)
		scoreJob(t, pool, scored, userID, `[]`)
		scoreJob(t, pool, scoredByOther, otherUserID, `[]`)
		scoreJob(t, pool, otherCompany, userID, `[]`)

		page, err := st.Page(t.Context(), userID, dto.JobPageOptions{Limit: 10, Availability: "open", CompanyID: acme, ScoredOnly: true})
		if err != nil {
			t.Fatalf("Page() err = %v", err)
		}
		if diff := cmp.Diff([]string{scored}, jobIDs(page.Items)); diff != "" {
			t.Errorf("Page(scored) ids (-want +got):\n%s", diff)
		}
	})
}

func TestSaveCanonical(t *testing.T) {
	const boardA = "11111111-1111-1111-1111-111111111111"
	const boardB = "22222222-2222-2222-2222-222222222222"

	t.Run("URL aliases and replays keep one job", func(t *testing.T) {
		st, _ := newStore(t)
		first := baseJob
		first.BoardID = boardA
		first.ProviderPostingID = "posting-1"
		first.URL = "https://example.com/jobs/1?ref=board"
		saved, status := saveJob(t, st, first)
		if status != "new" || saved.ID == "" {
			t.Fatalf("first save: status = %q, id = %q, want new with an id", status, saved.ID)
		}

		alias := first
		alias.URL = "https://example.com/jobs/1?ref=partner"
		for _, name := range []string{"alias", "replay"} {
			got, status := saveJob(t, st, alias)
			if status != "unchanged" || got.ID != saved.ID {
				t.Errorf("%s: status = %q, id = %q, want unchanged with id %q", name, status, got.ID, saved.ID)
			}
		}

		jobs, err := st.ListJobs(t.Context(), "")
		if err != nil {
			t.Fatalf("ListJobs() err = %v", err)
		}
		if len(jobs) != 1 {
			t.Fatalf("ListJobs() = %d jobs, want 1", len(jobs))
		}
		if jobs[0].BoardID != first.BoardID || jobs[0].ProviderPostingID != first.ProviderPostingID || jobs[0].ContentFingerprint == "" {
			t.Errorf("canonical identity missing from read: %+v", jobs[0])
		}
	})

	t.Run("a conflicting board cannot claim the URL", func(t *testing.T) {
		st, _ := newStore(t)
		first := baseJob
		first.BoardID = boardA
		first.ProviderPostingID = "posting-1"
		saveJob(t, st, first)

		conflict := baseJob
		conflict.BoardID = boardB
		conflict.ProviderPostingID = "posting-2"
		if _, _, err := st.SaveCanonical(t.Context(), conflict); !errors.Is(err, store.ErrCanonicalConflict) {
			t.Fatalf("SaveCanonical(conflict) err = %v, want ErrCanonicalConflict", err)
		}
	})

	t.Run("content change updates, and a distinct board is a distinct job", func(t *testing.T) {
		st, _ := newStore(t)
		first := baseJob
		first.BoardID = boardA
		first.ProviderPostingID = "posting-1"
		saved, _ := saveJob(t, st, first)

		changed := first
		changed.Title = "Senior Engineer"
		updated, status := saveJob(t, st, changed)
		if status != "changed" || updated.ID != saved.ID {
			t.Errorf("changed: status = %q, id = %q, want changed with id %q", status, updated.ID, saved.ID)
		}
		replayed, status := saveJob(t, st, changed)
		if status != "unchanged" || replayed.ID != saved.ID || replayed.ContentFingerprint != updated.ContentFingerprint {
			t.Errorf("changed replay: status = %q, job = %+v, want unchanged and the same fingerprint", status, replayed)
		}

		other := first
		other.BoardID = boardB
		other.URL = "https://other.example.com/jobs/1"
		distinct, status := saveJob(t, st, other)
		if status != "new" || distinct.ID == saved.ID {
			t.Errorf("distinct board: status = %q, id = %q, want new with an id other than %q", status, distinct.ID, saved.ID)
		}
	})

	t.Run("prunes stale option answers on fingerprint change", func(t *testing.T) {
		st, pool, userID := newUserStore(t)
		jobA := baseJob
		saved, _ := saveJob(t, st, jobA)
		fpA := saved.ContentFingerprint
		jobB := jobA
		jobB.Title = "Staff Engineer"
		updatedB, status := saveJob(t, st, jobB)
		if status != "changed" {
			t.Fatalf("change to B: status = %q, want changed", status)
		}
		insertOptionAnswer(t, pool, saved.ID, fpA, "q-fpA")
		insertOptionAnswer(t, pool, saved.ID, updatedB.ContentFingerprint, "q-fpB")

		revertedA, status := saveJob(t, st, jobA)
		if status != "changed" || revertedA.ContentFingerprint != fpA {
			t.Fatalf("revert to A: status = %q, fingerprint = %q, want changed with %q", status, revertedA.ContentFingerprint, fpA)
		}

		if diff := cmp.Diff(map[string]int{fpA: 1}, optionAnswerFingerprints(t, pool, saved.ID)); diff != "" {
			t.Errorf("option_answers after prune (-want +got):\n%s", diff)
		}
		scoreJob(t, pool, saved.ID, userID, `[]`)
		var visible int
		err := pool.QueryRow(t.Context(),
			`SELECT count(*) FROM job_scores s
			 JOIN jobs j ON j.id = s.job_id
			 JOIN option_answers a ON a.job_id = j.id AND a.fingerprint = j.content_fingerprint
			 WHERE s.user_id = $1 AND a.question_hash = 'q-fpB'`, userID).Scan(&visible)
		if err != nil {
			t.Fatalf("count scoring-visible answers: %v", err)
		}
		if visible != 0 {
			t.Errorf("pruned answer q-fpB still visible to scoring, want treated as unknown")
		}
	})
}

func TestSaveCanonicalQueuesAnswerEffects(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, st *store.Store, pool *pgxpool.Pool, userID, companyID string)
		titles  []string
		tracked bool
		want    int
	}{
		{name: "one per content version", tracked: true, titles: []string{"Engineer", "Engineer", "Senior Engineer"}, want: 2},
		{name: "none when no user tracks the company", titles: []string{"Engineer"}, want: 0},
		{
			name:    "regardless of the user's exclusion filters",
			tracked: true,
			titles:  []string{"Engineer"},
			want:    1,
			setup: func(t *testing.T, _ *store.Store, pool *pgxpool.Pool, userID, _ string) {
				t.Helper()
				if _, err := pool.Exec(t.Context(),
					"INSERT INTO search_config (user_id, excluded_companies) VALUES ($1, $2)", userID, []string{"outbox-co"}); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, pool, userID := newUserStore(t)
			company := upsertCompany(t, st, dto.CompanyUpsert{Slug: "outbox-co", Name: "Outbox Co"})
			if tt.setup != nil {
				tt.setup(t, st, pool, userID, company.ID)
			}
			if tt.tracked {
				trackCompany(t, st, userID, company.ID, 360)
			}
			job := baseJob
			job.CompanySlug, job.CompanyID = company.Slug, company.ID
			var jobID string
			for _, title := range tt.titles {
				job.Title = title
				saved, _ := saveJob(t, st, job)
				jobID = saved.ID
			}
			if got := effectCountForJob(t, pool, jobID); got != tt.want {
				t.Errorf("queued answer effects = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSetCompanyTracking(t *testing.T) {
	st, pool, userID := newUserStore(t)
	company := upsertCompany(t, st, dto.CompanyUpsert{Slug: "backfill-co", Name: "Backfill Co"})
	job := baseJob
	job.CompanySlug = company.Slug
	saved, _ := saveJob(t, st, job)

	tracking, err := st.SetCompanyTracking(t.Context(), userID, company.ID, true, 120)
	if err != nil {
		t.Fatalf("SetCompanyTracking() err = %v", err)
	}
	if !tracking.Enabled || tracking.CheckIntervalMinutes != 120 {
		t.Errorf("tracking = %+v, want enabled every 120 minutes", tracking)
	}
	if count := effectCountForJob(t, pool, saved.ID); count == 0 {
		t.Errorf("queued answer effects = %d, want > 0", count)
	}
}

func TestCompanyBoards(t *testing.T) {
	t.Run("candidate board is listed but not verified", func(t *testing.T) {
		st, _ := newStore(t)
		ctx := t.Context()
		company := upsertCompany(t, st, dto.CompanyUpsert{Slug: "board-co", Name: "Board Co"})
		board, err := st.UpsertCandidateBoard(ctx, company.ID, "greenhouse", "board-co")
		if err != nil {
			t.Fatalf("UpsertCandidateBoard() err = %v", err)
		}
		if board.Status != dto.BoardCandidate {
			t.Errorf("status = %q, want candidate", board.Status)
		}
		boards, err := st.ListCompanyBoards(ctx, company.ID)
		if err != nil {
			t.Fatalf("ListCompanyBoards() err = %v", err)
		}
		if len(boards) != 1 || boards[0].ID != board.ID {
			t.Errorf("ListCompanyBoards() = %+v, want only %s", boards, board.ID)
		}
		if _, err := st.GetVerifiedBoardID(ctx, "greenhouse", "board-co"); !errors.Is(err, data.ErrNotFound) {
			t.Errorf("GetVerifiedBoardID(unverified) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("a board token belongs to one company", func(t *testing.T) {
		st, _ := newStore(t)
		ctx := t.Context()
		first := upsertCompany(t, st, dto.CompanyUpsert{Slug: "acme", Name: "Acme"})
		second := upsertCompany(t, st, dto.CompanyUpsert{Slug: "other", Name: "Other"})
		if _, err := st.UpsertCandidateBoard(ctx, first.ID, "greenhouse", "acme"); err != nil {
			t.Fatalf("UpsertCandidateBoard(first) err = %v", err)
		}
		if _, err := st.UpsertCandidateBoard(ctx, second.ID, "greenhouse", "acme"); !errors.Is(err, store.ErrBoardConflict) {
			t.Fatalf("UpsertCandidateBoard(second) err = %v, want ErrBoardConflict", err)
		}
		if _, err := st.UpsertCandidateBoard(ctx, first.ID, "ashby", "acme"); err != nil {
			t.Fatalf("UpsertCandidateBoard(ashby) err = %v", err)
		}
		boards, err := st.ListCompanyBoards(ctx, first.ID)
		if err != nil {
			t.Fatalf("ListCompanyBoards() err = %v", err)
		}
		if len(boards) != 2 {
			t.Errorf("ListCompanyBoards() = %+v, want two boards", boards)
		}
	})
}

func TestDeleteExpiredCandidates(t *testing.T) {
	st, pool, userID := newUserStore(t)
	ctx := t.Context()
	target := createTarget(t, st, userID, "linkedin", "expiry-search")
	saved := saveCards(t, st, target, dto.Job{URL: "https://example.com/candidate/expiring", Title: "Engineer", CompanySlug: "acme"})
	if _, err := pool.Exec(ctx, "UPDATE job_candidates SET expires_at = NOW() - INTERVAL '1 day' WHERE id = $1::uuid", saved[0].ID); err != nil {
		t.Fatal(err)
	}

	if err := st.DeleteExpiredCandidates(ctx); err != nil {
		t.Fatalf("DeleteExpiredCandidates() err = %v", err)
	}

	listed, err := st.ListForUser(ctx, userID, "", 10)
	if err != nil {
		t.Fatalf("ListForUser() err = %v", err)
	}
	if len(listed) != 0 {
		t.Errorf("ListForUser() after expiry = %+v, want none", listed)
	}
}

func TestListCompaniesToCrawl(t *testing.T) {
	st, _ := newStore(t)
	ctx := t.Context()
	company := upsertCompany(t, st, dto.CompanyUpsert{Slug: "crawl-co", Name: "Crawl Co", Domain: "crawl-co.example"})

	due, err := st.ListCompaniesToCrawl(ctx, 10)
	if err != nil {
		t.Fatalf("ListCompaniesToCrawl() err = %v", err)
	}
	if len(due) != 1 || due[0].ID != company.ID {
		t.Fatalf("due for crawl = %+v, want only %s", due, company.ID)
	}

	if err := st.TouchCompanyCrawled(ctx, company.ID); err != nil {
		t.Fatalf("TouchCompanyCrawled() err = %v", err)
	}
	due, err = st.ListCompaniesToCrawl(ctx, 10)
	if err != nil {
		t.Fatalf("ListCompaniesToCrawl() after touch err = %v", err)
	}
	if len(due) != 0 {
		t.Errorf("due for crawl after touch = %+v, want none", due)
	}
}

func TestRenameCompany(t *testing.T) {
	st, _ := newStore(t)
	ctx := t.Context()
	company := upsertCompany(t, st, dto.CompanyUpsert{Slug: "faculty", Name: "Faculty"})

	if err := st.RenameCompany(ctx, company.ID, "Faculty AI"); err != nil {
		t.Fatalf("RenameCompany() err = %v", err)
	}

	got, err := st.GetCompany(ctx, company.ID)
	if err != nil {
		t.Fatalf("GetCompany() err = %v", err)
	}
	if got.Name != "Faculty AI" || got.Slug != "faculty" {
		t.Errorf("company = {%q %q}, want name %q slug %q", got.Name, got.Slug, "Faculty AI", "faculty")
	}
}

func TestSourceTargetRuns(t *testing.T) {
	t.Run("recovery claims a stale run once", func(t *testing.T) {
		st, pool, userID := newUserStore(t)
		ctx := t.Context()
		target, err := st.CreateSourceTargetWithRun(ctx, userID, "linkedin", "recovery-search", true, nil)
		if err != nil {
			t.Fatalf("CreateSourceTargetWithRun() err = %v", err)
		}
		running, err := st.TransitionSourceTargetRun(ctx, target.ID, target.RunID, "running", "")
		if err != nil {
			t.Fatalf("Transition(running) err = %v", err)
		}
		if running.RunStatus != "running" {
			t.Errorf("run status = %q, want running", running.RunStatus)
		}

		if _, err := pool.Exec(ctx, "UPDATE source_targets SET updated_at = NOW() - INTERVAL '1 hour' WHERE id = $1::uuid", target.ID); err != nil {
			t.Fatal(err)
		}
		recoverable, err := st.ListRecoverableSourceTargets(ctx)
		if err != nil {
			t.Fatalf("ListRecoverableSourceTargets() err = %v", err)
		}
		if len(recoverable) != 1 || recoverable[0].ID != target.ID {
			t.Fatalf("recoverable = %+v, want only %s", recoverable, target.ID)
		}

		claimed, err := st.ClaimRecoverableSourceTarget(ctx, target.ID, target.RunID)
		if err != nil {
			t.Fatalf("ClaimRecoverableSourceTarget() err = %v", err)
		}
		if claimed.ID != target.ID {
			t.Errorf("claimed id = %q, want %q", claimed.ID, target.ID)
		}
		if _, err := st.ClaimRecoverableSourceTarget(ctx, target.ID, target.RunID); !errors.Is(err, data.ErrNotFound) {
			t.Errorf("second claim err = %v, want ErrNotFound", err)
		}
	})

	t.Run("a newer run fences a stale completion", func(t *testing.T) {
		st, _, userID := newUserStore(t)
		ctx := t.Context()
		target := createTarget(t, st, userID, "wis", "engineer")
		if _, err := st.UpdateSourceTarget(ctx, target.ID, userID, new(false), nil); err != nil {
			t.Fatalf("UpdateSourceTarget() err = %v", err)
		}
		first, err := st.StartSourceTargetRun(ctx, target.ID)
		if err != nil {
			t.Fatalf("StartSourceTargetRun(first) err = %v", err)
		}
		if !first.Enabled {
			t.Error("manual rerun did not enable the target")
		}
		second, err := st.StartSourceTargetRun(ctx, target.ID)
		if err != nil {
			t.Fatalf("StartSourceTargetRun(second) err = %v", err)
		}
		if first.RunID == "" || first.RunID == second.RunID {
			t.Fatalf("run ids = %q, %q, want two distinct ids", first.RunID, second.RunID)
		}

		if _, err := st.TransitionSourceTargetRun(ctx, target.ID, first.RunID, "succeeded", ""); !errors.Is(err, data.ErrNotFound) {
			t.Errorf("stale completion err = %v, want ErrNotFound", err)
		}
		current, err := st.TransitionSourceTargetRun(ctx, target.ID, second.RunID, "succeeded", "")
		if err != nil {
			t.Fatalf("current completion err = %v", err)
		}
		if current.RunStatus != "succeeded" {
			t.Errorf("run status = %q, want succeeded", current.RunStatus)
		}
	})
}

type boardEnv struct {
	st        *store.Store
	pool      *pgxpool.Pool
	board     dto.CompanyBoard
	companyID string
}

func boardFixture(t *testing.T) boardEnv {
	t.Helper()
	st, pool, userID := newUserStore(t)
	company := upsertCompany(t, st, dto.CompanyUpsert{Slug: "poll-co", Name: "Poll Co"})
	board, err := st.UpsertCandidateBoard(t.Context(), company.ID, "greenhouse", "poll-co")
	if err != nil {
		t.Fatalf("UpsertCandidateBoard() err = %v", err)
	}
	trackCompany(t, st, userID, company.ID, 60)
	return boardEnv{st: st, pool: pool, board: board, companyID: company.ID}
}

func verifiedBoardFixture(t *testing.T) boardEnv {
	t.Helper()
	env := boardFixture(t)
	if _, err := env.st.VerifyCompanyBoard(t.Context(), env.companyID, env.board.Source, env.board.BoardToken, "user_confirmed"); err != nil {
		t.Fatalf("VerifyCompanyBoard() err = %v", err)
	}
	return env
}

func (e boardEnv) job(n int) dto.Job {
	return dto.Job{
		Title: fmt.Sprintf("Role %d", n), URL: fmt.Sprintf("https://boards.greenhouse.io/poll-co/jobs/%d", n), Source: "greenhouse",
		CompanySlug: "poll-co", CompanyID: e.companyID, BoardID: e.board.ID, UpdatedAt: time.Now(),
	}
}

func (e boardEnv) claim(t *testing.T, manual bool) dto.BoardPoll {
	t.Helper()
	poll, err := e.st.ClaimBoard(t.Context(), e.board.ID, manual)
	if err != nil {
		t.Fatalf("ClaimBoard(manual=%t) err = %v", manual, err)
	}
	return poll
}

func (e boardEnv) complete(t *testing.T, poll dto.BoardPoll, jobs ...dto.Job) {
	t.Helper()
	if err := e.st.CompleteBoard(t.Context(), dto.BoardSnapshot{Poll: poll, Complete: true, Jobs: jobs}); err != nil {
		t.Fatalf("CompleteBoard() err = %v", err)
	}
}

func (e boardEnv) due(t *testing.T) []dto.BoardPoll {
	t.Helper()
	due, err := e.st.ListDueBoards(t.Context())
	if err != nil {
		t.Fatalf("ListDueBoards() err = %v", err)
	}
	return due
}

func TestBoardPolling(t *testing.T) {
	t.Run("a board is due only once verified, and a claim is exclusive", func(t *testing.T) {
		env := boardFixture(t)
		ctx := t.Context()
		if due := env.due(t); len(due) != 0 {
			t.Fatalf("candidate board due = %v, want none", due)
		}
		if _, err := env.st.VerifyCompanyBoard(ctx, env.companyID, env.board.Source, env.board.BoardToken, "user_confirmed"); err != nil {
			t.Fatal(err)
		}
		if due := env.due(t); len(due) != 1 || due[0].ID != env.board.ID {
			t.Fatalf("verified board due = %v, want %s", due, env.board.ID)
		}

		claim := env.claim(t, false)
		if _, err := env.st.ClaimBoard(ctx, env.board.ID, false); !errors.Is(err, store.ErrBoardClaimUnavailable) {
			t.Errorf("second ClaimBoard() err = %v, want ErrBoardClaimUnavailable", err)
		}
		job := env.job(1)
		saveJob(t, env.st, job)
		env.complete(t, claim, job)
		if err := env.st.CompleteBoard(ctx, dto.BoardSnapshot{Poll: claim, Complete: true, Jobs: []dto.Job{job}}); !errors.Is(err, store.ErrBoardClaimUnavailable) {
			t.Errorf("replayed CompleteBoard() err = %v, want ErrBoardClaimUnavailable", err)
		}
		if due := env.due(t); len(due) != 0 {
			t.Errorf("completed board due = %v, want none", due)
		}

		for n := 1; n <= 2; n++ {
			env.complete(t, env.claim(t, true))
			if got := jobClosed(t, env.pool, job.URL); got != (n == 2) {
				t.Errorf("after %d empty snapshots closed = %t, want %t", n, got, n == 2)
			}
		}
	})

	t.Run("a next-poll hint sets next_due_at and holds the board back", func(t *testing.T) {
		env := verifiedBoardFixture(t)
		ctx := t.Context()
		hint := 3 * 24 * time.Hour
		err := env.st.CompleteBoard(ctx, dto.BoardSnapshot{Poll: env.claim(t, false), Complete: true, NextPollIn: hint})
		if err != nil {
			t.Fatalf("CompleteBoard() err = %v", err)
		}
		var days float64
		if err := env.pool.QueryRow(ctx, "SELECT EXTRACT(EPOCH FROM next_due_at - NOW()) / 86400 FROM board_poll_state WHERE board_id = $1", env.board.ID).Scan(&days); err != nil {
			t.Fatal(err)
		}
		if days < 2.99 || days > 3 {
			t.Errorf("next_due_at is %.3f days out, want 3", days)
		}
		if _, err := env.pool.Exec(ctx, "UPDATE board_poll_state SET last_scheduled_at = NOW() - INTERVAL '30 days' WHERE board_id = $1", env.board.ID); err != nil {
			t.Fatal(err)
		}
		if due := env.due(t); len(due) != 0 {
			t.Errorf("hinted board due = %v, want none", due)
		}
		if _, err := env.st.ClaimBoard(ctx, env.board.ID, false); !errors.Is(err, store.ErrBoardClaimUnavailable) {
			t.Errorf("ClaimBoard() before hint elapsed err = %v, want ErrBoardClaimUnavailable", err)
		}
	})

	t.Run("failing a claim needs the active lease", func(t *testing.T) {
		env := verifiedBoardFixture(t)
		claim := env.claim(t, false)
		if err := env.st.FailBoard(t.Context(), claim); err != nil {
			t.Fatalf("FailBoard() err = %v", err)
		}
		if err := env.st.FailBoard(t.Context(), claim); !errors.Is(err, store.ErrBoardClaimUnavailable) {
			t.Errorf("stale FailBoard() err = %v, want ErrBoardClaimUnavailable", err)
		}
	})

	t.Run("omitted jobs close, reappearing ones reopen, and bad snapshots are rejected", func(t *testing.T) {
		env := verifiedBoardFixture(t)
		ctx := t.Context()
		jobs := []dto.Job{env.job(1), env.job(2)}
		for _, job := range jobs {
			saveJob(t, env.st, job)
		}
		claim := env.claim(t, false)

		if err := env.st.CompleteBoard(ctx, dto.BoardSnapshot{Poll: claim, Complete: false, Jobs: jobs}); err == nil {
			t.Error("partial snapshot accepted")
		}
		unsaved := dto.Job{URL: "https://boards.greenhouse.io/poll-co/jobs/missing"}
		if err := env.st.CompleteBoard(ctx, dto.BoardSnapshot{Poll: claim, Complete: true, Jobs: append(jobs, unsaved)}); err == nil {
			t.Error("snapshot with a job that was never ingested accepted")
		}
		var completed bool
		if err := env.pool.QueryRow(ctx, "SELECT last_completed_at IS NOT NULL FROM board_poll_state WHERE board_id = $1", env.board.ID).Scan(&completed); err != nil {
			t.Fatal(err)
		}
		if completed {
			t.Error("rejected snapshots advanced freshness")
		}

		env.complete(t, claim, jobs...)
		env.complete(t, env.claim(t, true), jobs[0])
		if !jobClosed(t, env.pool, jobs[1].URL) {
			t.Error("omitted job is still open")
		}
		env.complete(t, env.claim(t, true), jobs...)
		if jobClosed(t, env.pool, jobs[1].URL) {
			t.Error("reappeared job is still closed")
		}
		if due := env.due(t); len(due) != 0 {
			t.Errorf("manual checks shifted cadence: %v", due)
		}
	})

	t.Run("a stale claim cannot overwrite a newer check", func(t *testing.T) {
		env := verifiedBoardFixture(t)
		old := env.claim(t, false)
		if _, err := env.pool.Exec(t.Context(), "UPDATE board_poll_state SET lease_until = NOW() - INTERVAL '1 minute' WHERE board_id = $1", env.board.ID); err != nil {
			t.Fatal(err)
		}
		env.complete(t, env.claim(t, true))

		err := env.st.CompleteBoard(t.Context(), dto.BoardSnapshot{Poll: old, Complete: true})
		if !errors.Is(err, store.ErrBoardClaimUnavailable) {
			t.Errorf("stale CompleteBoard() err = %v, want ErrBoardClaimUnavailable", err)
		}
	})

	t.Run("a superseded board retires after two empty checks", func(t *testing.T) {
		env := verifiedBoardFixture(t)
		ctx := t.Context()
		if _, err := env.pool.Exec(ctx, "UPDATE company_boards SET superseded_at = NOW() WHERE id = $1", env.board.ID); err != nil {
			t.Fatal(err)
		}
		for range 2 {
			env.complete(t, env.claim(t, true))
		}
		boards, err := env.st.ListCompanyBoards(ctx, env.companyID)
		if err != nil {
			t.Fatalf("ListCompanyBoards() err = %v", err)
		}
		if len(boards) != 1 || boards[0].Status != dto.BoardRetired {
			t.Errorf("boards = %+v, want one retired board", boards)
		}
		active, err := env.st.ListActiveBoards(ctx)
		if err != nil {
			t.Fatalf("ListActiveBoards() err = %v", err)
		}
		if len(active) != 0 {
			t.Errorf("active boards = %v, want none", active)
		}
	})
}

func insertOptionAnswer(t *testing.T, pool *pgxpool.Pool, jobID, fingerprint, questionHash string) {
	t.Helper()
	_, err := pool.Exec(t.Context(),
		"INSERT INTO option_answers (job_id, fingerprint, question_hash, model, p_yes, p_no, p_not_stated, confidence) VALUES ($1, $2, $3, 'test-model', 0.5, 0.3, 0.2, 0.9)",
		jobID, fingerprint, questionHash)
	if err != nil {
		t.Fatalf("insert option_answer: %v", err)
	}
}

func optionAnswerFingerprints(t *testing.T, pool *pgxpool.Pool, jobID string) map[string]int {
	t.Helper()
	rows, err := pool.Query(t.Context(), "SELECT fingerprint FROM option_answers WHERE job_id = $1", jobID)
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

func TestUpsertCompany(t *testing.T) {
	tests := []struct {
		name   string
		first  dto.CompanyUpsert
		second dto.CompanyUpsert
		want   dto.Company
	}{
		{
			name:   "conflict fills in missing ats fields",
			first:  dto.CompanyUpsert{Slug: "acme", Name: "Acme"},
			second: dto.CompanyUpsert{Slug: "acme", Name: "Acme", ATSSource: "greenhouse", ATSToken: "acme"},
			want:   dto.Company{Slug: "acme", Name: "Acme", ATSSource: "greenhouse", ATSToken: "acme"},
		},
		{
			name:   "conflict never overwrites an existing ats board",
			first:  dto.CompanyUpsert{Slug: "acme", Name: "Acme", ATSSource: "greenhouse", ATSToken: "acme"},
			second: dto.CompanyUpsert{Slug: "acme", Name: "Acme", ATSSource: "lever", ATSToken: "acme-other"},
			want:   dto.Company{Slug: "acme", Name: "Acme", ATSSource: "greenhouse", ATSToken: "acme"},
		},
		{
			name:   "conflict fills in missing domain and linkedin id",
			first:  dto.CompanyUpsert{Slug: "acme", Name: "Acme"},
			second: dto.CompanyUpsert{Slug: "acme", Name: "Acme", Domain: "acme.com", LinkedInCompanyID: "12345"},
			want:   dto.Company{Slug: "acme", Name: "Acme", Domain: "acme.com", LinkedInCompanyID: "12345"},
		},
		{
			name:   "conflict never overwrites an existing domain or linkedin id",
			first:  dto.CompanyUpsert{Slug: "acme", Name: "Acme", Domain: "acme.com", LinkedInCompanyID: "12345"},
			second: dto.CompanyUpsert{Slug: "acme", Name: "Acme", Domain: "other.com", LinkedInCompanyID: "99999"},
			want:   dto.Company{Slug: "acme", Name: "Acme", Domain: "acme.com", LinkedInCompanyID: "12345"},
		},
	}
	ignore := cmpopts.IgnoreFields(dto.Company{}, "ID", "LastCrawledAt", "FirstSeenAt", "LastCheckedAt")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, _ := newStore(t)
			first := upsertCompany(t, st, tt.first)
			got := upsertCompany(t, st, tt.second)
			if got.ID != first.ID {
				t.Errorf("second upsert id = %q, want the first row %q", got.ID, first.ID)
			}
			if diff := cmp.Diff(tt.want, got, ignore); diff != "" {
				t.Errorf("UpsertCompany() (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetCompanyForUser(t *testing.T) {
	st, _, userID := newUserStore(t)
	if _, err := st.GetCompanyForUser(t.Context(), userID, "not-a-uuid"); !errors.Is(err, store.ErrInvalidID) {
		t.Fatalf("GetCompanyForUser(malformed) err = %v, want ErrInvalidID", err)
	}
}

func TestListCompaniesForUser(t *testing.T) {
	t.Run("shows tracked and untracked companies", func(t *testing.T) {
		st, _, userID := newUserStore(t)
		tracked := upsertCompany(t, st, dto.CompanyUpsert{Slug: "acme", Name: "Acme"})
		upsertCompany(t, st, dto.CompanyUpsert{Slug: "widgetco", Name: "Widgetco"})
		trackCompany(t, st, userID, tracked.ID, 180)

		companies, err := st.ListCompaniesForUser(t.Context(), userID)
		if err != nil {
			t.Fatalf("ListCompaniesForUser() err = %v", err)
		}
		type view struct {
			Tracked  bool
			Interval int
		}
		got := map[string]view{}
		for _, c := range companies {
			got[c.Slug] = view{c.Tracked, c.CheckIntervalMinutes}
		}
		want := map[string]view{"acme": {true, 180}, "widgetco": {false, 0}}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("ListCompaniesForUser() (-want +got):\n%s", diff)
		}
	})

	t.Run("tracking without a board is per user", func(t *testing.T) {
		st, pool := newStore(t)
		alice := pgtest.InsertUser(t, pool)
		bob := pgtest.InsertUser(t, pool)
		company := upsertCompany(t, st, dto.CompanyUpsert{Slug: "boardless", Name: "Boardless"})
		trackCompany(t, st, alice, company.ID, 180)
		if _, err := st.SetCompanyTracking(t.Context(), alice, company.ID, false, 0); err != nil {
			t.Fatal(err)
		}
		tests := []struct {
			name         string
			userID       string
			wantInterval int
		}{
			{name: "paused for the user who tracked it", userID: alice, wantInterval: 180},
			{name: "untouched for another user", userID: bob},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				companies, err := st.ListCompaniesForUser(t.Context(), tt.userID)
				if err != nil {
					t.Fatalf("ListCompaniesForUser() err = %v", err)
				}
				if len(companies) != 1 {
					t.Fatalf("ListCompaniesForUser() = %+v, want one company", companies)
				}
				if companies[0].Tracked || companies[0].CheckIntervalMinutes != tt.wantInterval {
					t.Errorf("company = %+v, want untracked with interval %d", companies[0], tt.wantInterval)
				}
			})
		}
	})

	t.Run("reports when the board was last checked", func(t *testing.T) {
		st, pool, userID := newUserStore(t)
		company := upsertCompany(t, st, dto.CompanyUpsert{Slug: "acme", Name: "Acme"})
		board, err := st.UpsertCandidateBoard(t.Context(), company.ID, "greenhouse", "acme")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := st.VerifyCompanyBoard(t.Context(), company.ID, "greenhouse", "acme", "test"); err != nil {
			t.Fatal(err)
		}
		completed := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
		if _, err := pool.Exec(t.Context(), `INSERT INTO board_poll_state (board_id, last_completed_at) VALUES ($1, $2)`, board.ID, completed); err != nil {
			t.Fatal(err)
		}

		companies, err := st.ListCompaniesForUser(t.Context(), userID)
		if err != nil {
			t.Fatalf("ListCompaniesForUser() err = %v", err)
		}
		if len(companies) != 1 || companies[0].LastCheckedAt == nil || !companies[0].LastCheckedAt.Equal(completed) {
			t.Errorf("ListCompaniesForUser() = %+v, want last check %v", companies, completed)
		}
	})
}

func TestCandidates(t *testing.T) {
	st, pool, userID := newUserStore(t)
	ctx := t.Context()
	target := createTarget(t, st, userID, "wis", "engineer")
	card := dto.Job{URL: "https://example.com/jobs/1#details", Title: "Senior Engineer", CompanySlug: "acme", Location: "London"}

	got := saveCards(t, st, target, card, card)
	if len(got) != 2 || got[0].ID != got[1].ID || got[0].URL != "https://example.com/jobs/1" {
		t.Fatalf("duplicate cards saved as %+v, want one candidate on the canonical URL", got)
	}

	if _, err := pool.Exec(ctx, "UPDATE job_candidates SET expires_at = NOW() - INTERVAL '1 second'"); err != nil {
		t.Fatal(err)
	}
	retained, err := st.ListForUser(ctx, userID, "", 100)
	if err != nil {
		t.Fatalf("ListForUser(expired) err = %v", err)
	}
	if len(retained) != 0 {
		t.Fatalf("expired candidate remained queryable: %+v", retained)
	}

	saveCards(t, st, target, card)
	retained, err = st.ListForUser(ctx, userID, "", 100)
	if err != nil {
		t.Fatalf("ListForUser(rediscovered) err = %v", err)
	}
	if len(retained) != 1 {
		t.Fatalf("rediscovered candidate missing: %+v", retained)
	}
	if retained[0].Card.Source != target.Source {
		t.Errorf("candidate source = %q, want %q", retained[0].Card.Source, target.Source)
	}

	id := got[0].ID
	version := time.Now().UTC()
	assess := []struct {
		name          string
		at            time.Time
		relevant      bool
		wantRequested bool
		markPending   bool
	}{
		{name: "a rejected assessment requests no detail", at: version},
		{name: "a newly relevant candidate requests detail", at: version.Add(time.Second), relevant: true, wantRequested: true, markPending: true},
		{name: "a pending detail is not requested again", at: version.Add(2 * time.Second), relevant: true},
	}
	for _, tt := range assess {
		requested, err := st.Assess(ctx, id, userID, tt.at, tt.relevant)
		if err != nil {
			t.Fatalf("%s: Assess() err = %v", tt.name, err)
		}
		if requested != tt.wantRequested {
			t.Errorf("%s: Assess() = %t, want %t", tt.name, requested, tt.wantRequested)
		}
		if tt.markPending {
			if err := st.MarkDetailPending(ctx, id); err != nil {
				t.Fatalf("MarkDetailPending() err = %v", err)
			}
		}
	}
	var assessments int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM candidate_assessments WHERE candidate_id = $1 AND user_id = $2", id, userID).Scan(&assessments); err != nil {
		t.Fatal(err)
	}
	if assessments != 1 {
		t.Errorf("assessment rows = %d, want one latest row", assessments)
	}
}

func TestListTrackedCompaniesForUser(t *testing.T) {
	st, pool, userID := newUserStore(t)
	otherID := pgtest.InsertUser(t, pool)
	company := upsertCompany(t, st, dto.CompanyUpsert{Slug: "relevant-co", Name: "Relevant Co"})
	trackCompany(t, st, userID, company.ID, 180)

	now := time.Now()
	scoredOpen := insertJob(t, pool, company.ID, 1, now, false)
	scoreJob(t, pool, scoredOpen, userID, `[]`)
	blocked := insertJob(t, pool, company.ID, 2, now, false)
	scoreJob(t, pool, blocked, userID, `[{"effect":"blocked"}]`)
	closedScored := insertJob(t, pool, company.ID, 3, now, true)
	scoreJob(t, pool, closedScored, userID, `[]`)
	scoredByOther := insertJob(t, pool, company.ID, 4, now, false)
	scoreJob(t, pool, scoredByOther, otherID, `[]`)
	insertJob(t, pool, company.ID, 5, now, false)

	got, err := st.ListTrackedCompaniesForUser(t.Context(), userID)
	if err != nil {
		t.Fatalf("ListTrackedCompaniesForUser() err = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListTrackedCompaniesForUser() = %+v, want one company", got)
	}
	if got[0].OpenJobs != 4 || got[0].RelevantJobs != 1 {
		t.Errorf("open, relevant = %d, %d, want 4, 1", got[0].OpenJobs, got[0].RelevantJobs)
	}
}

func TestDeleteExpiredFetches(t *testing.T) {
	st, pool := newStore(t)
	ctx := t.Context()
	stale := dto.CachedResponse{URL: "https://example.com/stale", Status: 200, Header: http.Header{}, Body: []byte("a")}
	fresh := dto.CachedResponse{URL: "https://example.com/fresh", Status: 200, Header: http.Header{}, Body: []byte("b")}
	for _, resp := range []dto.CachedResponse{stale, fresh} {
		if err := st.PutFetch(ctx, resp); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, "UPDATE fetch_cache SET fetched_at = NOW() - INTERVAL '8 days' WHERE url = $1", stale.URL); err != nil {
		t.Fatal(err)
	}

	if err := st.DeleteExpiredFetches(ctx); err != nil {
		t.Fatalf("DeleteExpiredFetches() err = %v", err)
	}

	if _, ok, _ := st.LookupFetch(ctx, stale.URL); ok {
		t.Error("LookupFetch(stale) hit, want miss")
	}
	if _, ok, _ := st.LookupFetch(ctx, fresh.URL); !ok {
		t.Error("LookupFetch(fresh) miss, want hit")
	}
}

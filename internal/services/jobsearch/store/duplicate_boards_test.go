package store_test

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
)

func verifiedBoard(t *testing.T, st *store.Store, userID, slug, source, token string) dto.CompanyBoard {
	t.Helper()
	company := upsertCompany(t, st, dto.CompanyUpsert{Slug: slug, Name: slug})
	board, err := st.UpsertCandidateBoard(t.Context(), company.ID, source, token)
	if err != nil {
		t.Fatalf("UpsertCandidateBoard(%s, %s) err = %v", source, token, err)
	}
	if _, err := st.VerifyCompanyBoard(t.Context(), company.ID, source, token, "test", ""); err != nil {
		t.Fatalf("VerifyCompanyBoard(%s, %s) err = %v", source, token, err)
	}
	trackCompany(t, st, userID, company.ID, 60)
	board.CompanyID = company.ID
	return board
}

func boardJob(board dto.CompanyBoard, source, url, posting string) dto.Job {
	return dto.Job{
		Title: "Role " + posting, Location: "London", URL: url, Source: source, CompanySlug: "slug-" + board.ID,
		CompanyID: board.CompanyID, BoardID: board.ID, ProviderPostingID: posting,
	}
}

func pollBoard(t *testing.T, st *store.Store, board dto.CompanyBoard, jobs ...dto.Job) []dto.JobChange {
	t.Helper()
	poll, err := st.ClaimBoard(t.Context(), board.ID, true)
	if err != nil {
		t.Fatalf("ClaimBoard(%s) err = %v", board.BoardToken, err)
	}
	changes, err := st.CompleteBoard(t.Context(), dto.BoardSnapshot{Poll: poll, Complete: true, Jobs: jobs, Reported: len(jobs)})
	if err != nil {
		t.Fatalf("CompleteBoard(%s) err = %v", board.BoardToken, err)
	}
	return changes
}

func boardColumn(t *testing.T, pool *pgxpool.Pool, query, id string) string {
	t.Helper()
	var got string
	if err := pool.QueryRow(t.Context(), query, id).Scan(&got); err != nil {
		t.Fatal(err)
	}
	return got
}

func TestStore_UpsertCandidateBoardCaseSensitiveSourceReturnsExistingBoard(t *testing.T) {
	t.Parallel()
	st, _ := newStore(t)
	company := upsertCompany(t, st, dto.CompanyUpsert{Slug: "foo", Name: "Foo"})
	first, err := st.UpsertCandidateBoard(t.Context(), company.ID, "greenhouse", "Foo")
	if err != nil {
		t.Fatal(err)
	}
	second, err := st.UpsertCandidateBoard(t.Context(), company.ID, "greenhouse", "foo")
	if err != nil || second.ID != first.ID {
		t.Fatalf("UpsertCandidateBoard(foo) = %+v, %v, want board %s", second, err, first.ID)
	}
}

func TestSaveCanonical_SameSourceOwnerBoard(t *testing.T) {
	t.Parallel()
	st, pool, userID := newUserStore(t)
	owner := verifiedBoard(t, st, userID, "cfc", "pinpoint", "cfc")
	alias := verifiedBoard(t, st, userID, "cfcunderwriting", "pinpoint", "cfcunderwriting")

	shared, _ := saveJob(t, st, boardJob(owner, "pinpoint", "https://cfc.pinpointhq.com/postings/1", "1"))
	own, _ := saveJob(t, st, boardJob(alias, "pinpoint", "https://cfcunderwriting.pinpointhq.com/postings/2", "2"))
	pollBoard(t, st, owner, shared)
	pollBoard(t, st, alias, own)

	dup := boardJob(alias, "pinpoint", "https://cfc.pinpointhq.com/postings/1", "1")
	got, status, err := st.SaveCanonical(t.Context(), dup)
	if err != nil || status != "unchanged" || got.ID != shared.ID {
		t.Fatalf("SaveCanonical() = %q, %q, %v, want unchanged on job %s", got.ID, status, err, shared.ID)
	}

	if s := boardColumn(t, pool, `SELECT status FROM company_boards WHERE id = $1`, alias.ID); s != "retired" {
		t.Errorf("alias Board status = %s, want retired", s)
	}
	if b := boardColumn(t, pool, `SELECT primary_board_id::text FROM jobs WHERE id = $1`, own.ID); b != owner.ID {
		t.Errorf("re-homed job primary_board_id = %s, want owner %s", b, owner.ID)
	}

	changes := pollBoard(t, st, owner, shared)
	want := dto.JobChange{JobID: own.ID, Kind: dto.JobChangeRemoved}
	if len(changes) != 1 || changes[0] != want {
		t.Errorf("owner poll changes = %+v, want %+v", changes, want)
	}
}

func TestSaveCanonical_OtherSourceOwnerBoard(t *testing.T) {
	t.Parallel()
	st, pool, userID := newUserStore(t)
	owner := verifiedBoard(t, st, userID, "kraken", "ashby", "krakentech")
	other := verifiedBoard(t, st, userID, "kraken-wttj", "wttj", "Kraken-tech")
	first, _ := saveJob(t, st, boardJob(owner, "ashby", "https://jobs.ashbyhq.com/krakentech/1", "1"))

	got, status, err := st.SaveCanonical(t.Context(), boardJob(other, "wttj", "https://jobs.ashbyhq.com/krakentech/1", "w1"))
	if err != nil || status != "unchanged" || got.ID != first.ID {
		t.Fatalf("SaveCanonical() = %q, %q, %v, want unchanged on job %s", got.ID, status, err, first.ID)
	}
	if s := boardColumn(t, pool, `SELECT status FROM company_boards WHERE id = $1`, other.ID); s != "verified" {
		t.Errorf("other-source Board status = %s, want verified", s)
	}
	if n := jobCount(t, pool); n != 1 {
		t.Errorf("jobs = %d, want 1", n)
	}
}

func TestSaveCanonical_TrustedPostingReclaimsURLFromUntrustedAlias(t *testing.T) {
	t.Parallel()
	st, pool, userID := newUserStore(t)
	board := verifiedBoard(t, st, userID, "speechify", "greenhouse", "speechify")
	wrong, _ := saveJob(t, st, boardJob(board, "greenhouse", "https://boards.greenhouse.io/speechify/jobs/6229014004", "6229014004"))
	const contested = "https://boards.greenhouse.io/speechify/jobs/6229028004"
	if _, err := pool.Exec(t.Context(),
		`INSERT INTO job_urls (job_id, normalized_url, source) VALUES ($1, $2, 'hiringcafe')`, wrong.ID, contested); err != nil {
		t.Fatal(err)
	}

	got, status, err := st.SaveCanonical(t.Context(), boardJob(board, "greenhouse", contested, "6229028004"))
	if err != nil || status != "new" || got.ID == wrong.ID {
		t.Fatalf("SaveCanonical() = %q, %q, %v, want a new job apart from %s", got.ID, status, err, wrong.ID)
	}
	if owner := boardColumn(t, pool, `SELECT job_id::text FROM job_urls WHERE normalized_url = $1`, contested); owner != got.ID {
		t.Errorf("URL owner = %s, want %s", owner, got.ID)
	}
}

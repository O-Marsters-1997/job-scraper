package db_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func boardFixture(t *testing.T) (context.Context, dto.CompanyBoard, string) {
	t.Helper()
	truncateCompanies(t)
	ctx := context.Background()
	var userID string
	if err := testDB.Pool().QueryRow(ctx, "INSERT INTO users (username, password_hash) VALUES ('board-user', 'x') RETURNING id::text").Scan(&userID); err != nil {
		t.Fatal(err)
	}
	company, err := testDB.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "board-co", Name: "Board Co"})
	if err != nil {
		t.Fatal(err)
	}
	board, err := testDB.UpsertCandidateBoard(ctx, company.ID, "greenhouse", "board-co")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.SetCompanyTracking(ctx, userID, company.ID, true, 60); err != nil {
		t.Fatal(err)
	}
	return ctx, board, company.ID
}

func TestBoardPollDueClaimAndEmptyClosure(t *testing.T) {
	ctx, board, companyID := boardFixture(t)
	if due, err := testDB.ListDueBoards(ctx); err != nil || len(due) != 0 {
		t.Fatalf("candidate due=%v err=%v", due, err)
	}
	if _, err := testDB.VerifyCompanyBoard(ctx, companyID, board.Source, board.BoardToken, "user_confirmed"); err != nil {
		t.Fatal(err)
	}
	due, err := testDB.ListDueBoards(ctx)
	if err != nil || len(due) != 1 || due[0].ID != board.ID {
		t.Fatalf("verified due=%v err=%v", due, err)
	}
	claim, err := testDB.ClaimBoard(ctx, board.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.ClaimBoard(ctx, board.ID, false); !errors.Is(err, providers.ErrBoardClaimUnavailable) {
		t.Fatalf("second claim=%v", err)
	}
	job := dto.Job{Title: "Engineer", URL: "https://boards.greenhouse.io/board-co/jobs/1", Source: "greenhouse", CompanySlug: "board-co", CompanyID: companyID, BoardID: board.ID, UpdatedAt: time.Now()}
	if _, _, err := testDB.SaveCanonical(ctx, job); err != nil {
		t.Fatal(err)
	}
	if err := testDB.CompleteBoard(ctx, dto.BoardSnapshot{Poll: claim, Complete: true, Jobs: []dto.Job{job}}); err != nil {
		t.Fatal(err)
	}
	boards, err := testDB.ListCompanyBoards(ctx, companyID)
	if err != nil || len(boards) != 1 || boards[0].LastCompletedAt == nil {
		t.Fatalf("company board check=%v err=%v", boards, err)
	}
	if err := testDB.CompleteBoard(ctx, dto.BoardSnapshot{Poll: claim, Complete: true, Jobs: []dto.Job{job}}); !errors.Is(err, providers.ErrBoardClaimUnavailable) {
		t.Fatalf("replay=%v", err)
	}
	if due, err := testDB.ListDueBoards(ctx); err != nil || len(due) != 0 {
		t.Fatalf("completed due=%v err=%v", due, err)
	}
	if _, err := testDB.ClaimBoard(ctx, board.ID, false); !errors.Is(err, providers.ErrBoardClaimUnavailable) {
		t.Fatalf("scheduled claim after completion=%v", err)
	}
	for n := 1; n <= 2; n++ {
		empty, err := testDB.ClaimBoard(ctx, board.ID, true)
		if err != nil {
			t.Fatal(err)
		}
		if err := testDB.CompleteBoard(ctx, dto.BoardSnapshot{Poll: empty, Complete: true}); err != nil {
			t.Fatal(err)
		}
		var closed bool
		if err := testDB.Pool().QueryRow(ctx, "SELECT closed_at IS NOT NULL FROM jobs WHERE url = $1", job.URL).Scan(&closed); err != nil {
			t.Fatal(err)
		}
		if closed != (n == 2) {
			t.Fatalf("after %d empty snapshots closed=%t", n, closed)
		}
	}
}

func TestBoardPollOmissionReopenAndRejectedCompletion(t *testing.T) {
	ctx, board, companyID := boardFixture(t)
	if _, err := testDB.VerifyCompanyBoard(ctx, companyID, board.Source, board.BoardToken, "user_confirmed"); err != nil {
		t.Fatal(err)
	}
	jobs := []dto.Job{
		{Title: "Engineer", URL: "https://boards.greenhouse.io/board-co/jobs/1", Source: "greenhouse", CompanySlug: "board-co", CompanyID: companyID, BoardID: board.ID, UpdatedAt: time.Now()},
		{Title: "Designer", URL: "https://boards.greenhouse.io/board-co/jobs/2", Source: "greenhouse", CompanySlug: "board-co", CompanyID: companyID, BoardID: board.ID, UpdatedAt: time.Now()},
	}
	for _, job := range jobs {
		if _, _, err := testDB.SaveCanonical(ctx, job); err != nil {
			t.Fatal(err)
		}
	}
	claim, err := testDB.ClaimBoard(ctx, board.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := testDB.CompleteBoard(ctx, dto.BoardSnapshot{Poll: claim, Complete: false, Jobs: jobs}); err == nil {
		t.Fatal("partial snapshot accepted")
	}
	if err := testDB.CompleteBoard(ctx, dto.BoardSnapshot{Poll: claim, Complete: true, Jobs: append(jobs, dto.Job{URL: "https://boards.greenhouse.io/board-co/jobs/missing"})}); err == nil {
		t.Fatal("missing ingest accepted")
	}
	var completed bool
	if err := testDB.Pool().QueryRow(ctx, "SELECT last_completed_at IS NOT NULL FROM board_poll_state WHERE board_id = $1", board.ID).Scan(&completed); err != nil || completed {
		t.Fatalf("failed snapshot advanced freshness: %t, %v", completed, err)
	}
	if err := testDB.CompleteBoard(ctx, dto.BoardSnapshot{Poll: claim, Complete: true, Jobs: jobs}); err != nil {
		t.Fatal(err)
	}
	second, err := testDB.ClaimBoard(ctx, board.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := testDB.CompleteBoard(ctx, dto.BoardSnapshot{Poll: second, Complete: true, Jobs: jobs[:1]}); err != nil {
		t.Fatal(err)
	}
	var closed bool
	if err := testDB.Pool().QueryRow(ctx, "SELECT closed_at IS NOT NULL FROM jobs WHERE url = $1", jobs[1].URL).Scan(&closed); err != nil || !closed {
		t.Fatalf("omitted job closed=%t err=%v", closed, err)
	}
	third, err := testDB.ClaimBoard(ctx, board.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := testDB.CompleteBoard(ctx, dto.BoardSnapshot{Poll: third, Complete: true, Jobs: jobs}); err != nil {
		t.Fatal(err)
	}
	if err := testDB.Pool().QueryRow(ctx, "SELECT closed_at IS NOT NULL FROM jobs WHERE url = $1", jobs[1].URL).Scan(&closed); err != nil || closed {
		t.Fatalf("reappeared job closed=%t err=%v", closed, err)
	}
	if due, err := testDB.ListDueBoards(ctx); err != nil || len(due) != 0 {
		t.Fatalf("manual checks shifted cadence: %v %v", due, err)
	}
}

func TestBoardPollStaleClaimCannotOverwriteNewerCheck(t *testing.T) {
	ctx, board, companyID := boardFixture(t)
	if _, err := testDB.VerifyCompanyBoard(ctx, companyID, board.Source, board.BoardToken, "user_confirmed"); err != nil {
		t.Fatal(err)
	}
	old, err := testDB.ClaimBoard(ctx, board.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Pool().Exec(ctx, "UPDATE board_poll_state SET lease_until = NOW() - INTERVAL '1 minute' WHERE board_id = $1", board.ID); err != nil {
		t.Fatal(err)
	}
	newer, err := testDB.ClaimBoard(ctx, board.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := testDB.CompleteBoard(ctx, dto.BoardSnapshot{Poll: newer, Complete: true}); err != nil {
		t.Fatal(err)
	}
	if err := testDB.CompleteBoard(ctx, dto.BoardSnapshot{Poll: old, Complete: true}); !errors.Is(err, providers.ErrBoardClaimUnavailable) {
		t.Fatalf("stale completion=%v", err)
	}
}

func TestBoardPollRetiresSupersededBoardAfterTwoEmptyChecks(t *testing.T) {
	ctx, board, companyID := boardFixture(t)
	if _, err := testDB.VerifyCompanyBoard(ctx, companyID, board.Source, board.BoardToken, "user_confirmed"); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Pool().Exec(ctx, "UPDATE company_boards SET superseded_at = NOW() WHERE id = $1", board.ID); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		claim, err := testDB.ClaimBoard(ctx, board.ID, true)
		if err != nil {
			t.Fatal(err)
		}
		if err := testDB.CompleteBoard(ctx, dto.BoardSnapshot{Poll: claim, Complete: true}); err != nil {
			t.Fatal(err)
		}
	}
	boards, err := testDB.ListCompanyBoards(ctx, companyID)
	if err != nil || len(boards) != 1 || boards[0].Status != dto.BoardRetired {
		t.Fatalf("board retirement=%v err=%v", boards, err)
	}
	if due, err := testDB.ListActiveBoards(ctx); err != nil || len(due) != 0 {
		t.Fatalf("retired board active=%v err=%v", due, err)
	}
}

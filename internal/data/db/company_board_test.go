package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestCompanyBoardAssociation(t *testing.T) {
	truncateCompanies(t)
	ctx := context.Background()
	first, err := testDB.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := testDB.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "other", Name: "Other"})
	if err != nil {
		t.Fatal(err)
	}
	board, err := testDB.UpsertCandidateBoard(ctx, first.ID, "greenhouse", "acme")
	if err != nil || board.Status != dto.BoardCandidate {
		t.Fatalf("candidate: %+v %v", board, err)
	}
	if _, err := testDB.UpsertCandidateBoard(ctx, second.ID, "greenhouse", "acme"); !errors.Is(err, providers.ErrBoardConflict) {
		t.Fatalf("want conflict, got %v", err)
	}
	board, err = testDB.VerifyCompanyBoard(ctx, first.ID, "greenhouse", "acme", "user_confirmed")
	if err != nil || board.Status != dto.BoardVerified || board.VerifiedAt == nil {
		t.Fatalf("verify: %+v %v", board, err)
	}
	boards, err := testDB.ListCompanyBoards(ctx, first.ID)
	if err != nil || len(boards) != 1 || boards[0].ID != board.ID {
		t.Fatalf("list: %+v %v", boards, err)
	}
	secondBoard, err := testDB.UpsertCandidateBoard(ctx, first.ID, "ashby", "acme")
	if err != nil || secondBoard.Status != dto.BoardCandidate {
		t.Fatalf("second board: %+v %v", secondBoard, err)
	}
	boards, err = testDB.ListCompanyBoards(ctx, first.ID)
	if err != nil || len(boards) != 2 {
		t.Fatalf("want two boards: %+v %v", boards, err)
	}
}

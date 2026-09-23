package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func fromCompanyBoard(row pgsqlc.CompanyBoard) dto.CompanyBoard {
	board := dto.CompanyBoard{
		ID:                 row.ID.String(),
		CompanyID:          row.CompanyID.String(),
		Source:             row.Source,
		BoardToken:         row.BoardToken,
		Status:             dto.BoardStatus(row.Status),
		VerificationMethod: row.VerificationMethod.String,
		CreatedAt:          row.CreatedAt.Time,
	}
	if row.VerifiedAt.Valid {
		board.VerifiedAt = &row.VerifiedAt.Time
	}
	if row.LastLinkedAt.Valid {
		board.LastLinkedAt = &row.LastLinkedAt.Time
	}
	if row.RetiredAt.Valid {
		board.RetiredAt = &row.RetiredAt.Time
	}
	return board
}

func (db *DB) ListCompanyBoards(ctx context.Context, companyID string) ([]dto.CompanyBoard, error) {
	id, err := parseUUID(companyID)
	if err != nil {
		return nil, err
	}
	rows, err := db.queries.ListCompanyBoards(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("db.ListCompanyBoards: %w", err)
	}
	boards := make([]dto.CompanyBoard, len(rows))
	for i, row := range rows {
		boards[i] = fromCompanyBoard(row)
	}
	return boards, nil
}

func (db *DB) UpsertCandidateBoard(ctx context.Context, companyID, source, token string) (dto.CompanyBoard, error) {
	id, err := parseUUID(companyID)
	if err != nil {
		return dto.CompanyBoard{}, err
	}
	row, err := db.queries.UpsertCandidateBoard(ctx, pgsqlc.UpsertCandidateBoardParams{
		CompanyID: id, Source: source, BoardToken: token,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.CompanyBoard{}, providers.ErrBoardConflict
	}
	if err != nil {
		return dto.CompanyBoard{}, fmt.Errorf("db.UpsertCandidateBoard: %w", err)
	}
	return fromCompanyBoard(row), nil
}

func (db *DB) VerifyCompanyBoard(ctx context.Context, companyID, source, token, method string) (dto.CompanyBoard, error) {
	id, err := parseUUID(companyID)
	if err != nil {
		return dto.CompanyBoard{}, err
	}
	row, err := db.queries.VerifyCompanyBoard(ctx, pgsqlc.VerifyCompanyBoardParams{
		CompanyID: id, Source: source, BoardToken: token,
		VerificationMethod: pgtype.Text{String: method, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.CompanyBoard{}, providers.ErrNotFound
	}
	if err != nil {
		return dto.CompanyBoard{}, fmt.Errorf("db.VerifyCompanyBoard: %w", err)
	}
	return fromCompanyBoard(row), nil
}

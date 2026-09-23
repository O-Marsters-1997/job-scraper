package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

const boardColumns = "id, company_id, source, board_token, status, COALESCE(verification_method, ''), verified_at, last_linked_at, retired_at, created_at"

func scanBoard(row pgx.Row) (dto.CompanyBoard, error) {
	var board dto.CompanyBoard
	var id, companyID pgtype.UUID
	var status string
	var verifiedAt, lastLinkedAt, retiredAt *time.Time
	err := row.Scan(&id, &companyID, &board.Source, &board.BoardToken, &status, &board.VerificationMethod, &verifiedAt, &lastLinkedAt, &retiredAt, &board.CreatedAt)
	if err != nil {
		return dto.CompanyBoard{}, err
	}
	board.ID = uuidString(id.Bytes)
	board.CompanyID = uuidString(companyID.Bytes)
	board.Status = dto.BoardStatus(status)
	board.VerifiedAt = verifiedAt
	board.LastLinkedAt = lastLinkedAt
	board.RetiredAt = retiredAt
	return board, nil
}

func uuidString(id [16]byte) string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", id[:4], id[4:6], id[6:8], id[8:10], id[10:])
}

func (db *DB) ListCompanyBoards(ctx context.Context, companyID string) ([]dto.CompanyBoard, error) {
	id, err := parseUUID(companyID)
	if err != nil {
		return nil, err
	}
	rows, err := db.pool.Query(ctx, "SELECT "+boardColumns+" FROM company_boards WHERE company_id = $1 ORDER BY created_at, id", id)
	if err != nil {
		return nil, fmt.Errorf("db.ListCompanyBoards: %w", err)
	}
	defer rows.Close()
	boards := []dto.CompanyBoard{}
	for rows.Next() {
		board, err := scanBoard(rows)
		if err != nil {
			return nil, fmt.Errorf("db.ListCompanyBoards: %w", err)
		}
		boards = append(boards, board)
	}
	return boards, rows.Err()
}

func (db *DB) UpsertCandidateBoard(ctx context.Context, companyID, source, token string) (dto.CompanyBoard, error) {
	id, err := parseUUID(companyID)
	if err != nil {
		return dto.CompanyBoard{}, err
	}
	board, err := scanBoard(db.pool.QueryRow(ctx, "INSERT INTO company_boards (company_id, source, board_token) VALUES ($1, $2, $3) ON CONFLICT (source, board_token) DO UPDATE SET source = EXCLUDED.source RETURNING "+boardColumns, id, source, token))
	if err != nil {
		return dto.CompanyBoard{}, fmt.Errorf("db.UpsertCandidateBoard: %w", err)
	}
	if board.CompanyID != companyID {
		return dto.CompanyBoard{}, providers.ErrBoardConflict
	}
	return board, nil
}

func (db *DB) VerifyCompanyBoard(ctx context.Context, companyID, source, token, method string) (dto.CompanyBoard, error) {
	id, err := parseUUID(companyID)
	if err != nil {
		return dto.CompanyBoard{}, err
	}
	board, err := scanBoard(db.pool.QueryRow(ctx, "UPDATE company_boards SET status = 'verified', verification_method = $4, verified_at = NOW() WHERE company_id = $1 AND source = $2 AND board_token = $3 AND status = 'candidate' RETURNING "+boardColumns, id, source, token, method))
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.CompanyBoard{}, providers.ErrNotFound
	}
	if err != nil {
		return dto.CompanyBoard{}, fmt.Errorf("db.VerifyCompanyBoard: %w", err)
	}
	return board, nil
}

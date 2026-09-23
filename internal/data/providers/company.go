package providers

import (
	"context"
	"errors"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

var ErrBoardConflict = errors.New("board belongs to another company")
var ErrBoardClaimUnavailable = errors.New("board claim unavailable")

type CompanyProvider interface {
	UpsertCompany(ctx context.Context, c dto.CompanyUpsert) (dto.Company, error)
	GetCompany(ctx context.Context, id string) (dto.Company, error)
	ListCompaniesForUser(ctx context.Context, userID string) ([]dto.Company, error)
	SetCompanyTracking(ctx context.Context, userID, companyID string, enabled bool, checkIntervalMinutes int) (dto.CompanyTracking, error)
	ListCompanyBoards(ctx context.Context, companyID string) ([]dto.CompanyBoard, error)
	UpsertCandidateBoard(ctx context.Context, companyID, source, token string) (dto.CompanyBoard, error)
	VerifyCompanyBoard(ctx context.Context, companyID, source, token, method string) (dto.CompanyBoard, error)
}

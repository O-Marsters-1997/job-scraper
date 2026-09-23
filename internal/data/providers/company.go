package providers

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type CompanyProvider interface {
	UpsertCompany(ctx context.Context, c dto.CompanyUpsert) (dto.Company, error)
	GetCompany(ctx context.Context, id string) (dto.Company, error)
	ListCompaniesForUser(ctx context.Context, userID string) ([]dto.Company, error)
	SetCompanyTracking(ctx context.Context, userID, companyID string, enabled bool, checkIntervalMinutes int) (dto.CompanyTracking, error)
}

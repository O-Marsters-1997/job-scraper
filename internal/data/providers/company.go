package providers

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type CompanyProvider interface {
	UpsertCompany(ctx context.Context, slug, name, atsSource, atsToken string) (dto.Company, error)
	GetCompany(ctx context.Context, id string) (dto.Company, error)
	ListCompaniesForUser(ctx context.Context, userID string) ([]dto.Company, error)
}

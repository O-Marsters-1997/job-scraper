package jobsearch

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func (s *Service) SetCompanyKind(ctx context.Context, userID string, in dto.SetCompanyKindInput) (dto.Company, error) {
	panic("not implemented")
}

func (m *Module) SuggestCompanyKind(ctx context.Context, companyID string, kind dto.CompanyKind) (bool, error) {
	panic("not implemented")
}

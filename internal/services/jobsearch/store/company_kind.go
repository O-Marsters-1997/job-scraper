package store

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func (s *Store) SetCompanyKind(ctx context.Context, companyID string, kind dto.CompanyKind) error {
	panic("not implemented")
}

func (s *Store) SuggestCompanyKind(ctx context.Context, companyID string, kind dto.CompanyKind) (bool, error) {
	panic("not implemented")
}

package jobsearch

import (
	"context"
	"errors"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

// ExcludeCompany rules the Company out for userID: it is dismissed, so its
// Boards stop polling and discovery stops re-tracking it, and its name joins
// ExcludedCompanies. Repeating it changes nothing; Grades are untouched.
func (m *Module) ExcludeCompany(ctx context.Context, userID string, in dto.ExcludeCompanyInput) (dto.CompanyExclusion, error) {
	company, err := m.store.GetCompany(ctx, in.CompanyID)
	if err != nil {
		return dto.CompanyExclusion{}, err
	}
	added, err := m.scoring.ExcludeCompany(ctx, userID, company.Name)
	if err != nil {
		return dto.CompanyExclusion{}, err
	}
	if err := m.setReview(ctx, userID, company.ID, "dismissed"); err != nil {
		return dto.CompanyExclusion{}, err
	}
	return dto.CompanyExclusion{Name: company.Name, Added: added}, nil
}

// UnexcludeCompany reverses ExcludeCompany: the review state returns to new,
// and the name leaves ExcludedCompanies only when in.RemoveName says the
// exclusion added it.
func (m *Module) UnexcludeCompany(ctx context.Context, userID string, in dto.UnexcludeCompanyInput) (dto.CompanyExclusion, error) {
	company, err := m.store.GetCompany(ctx, in.CompanyID)
	if err != nil {
		return dto.CompanyExclusion{}, err
	}
	if in.RemoveName {
		if err := m.scoring.UnexcludeCompany(ctx, userID, company.Name); err != nil {
			return dto.CompanyExclusion{}, err
		}
	}
	if err := m.setReview(ctx, userID, company.ID, "new"); err != nil {
		return dto.CompanyExclusion{}, err
	}
	return dto.CompanyExclusion{Name: company.Name}, nil
}

func (m *Module) setReview(ctx context.Context, userID, companyID, state string) error {
	_, err := m.store.SetCompanyReviewState(ctx, userID, companyID, state)
	if errors.Is(err, data.ErrNotFound) {
		return nil
	}
	return err
}

package jobsearch

import (
	"context"
	"errors"

	"github.com/ollymarsters/job-scraper/internal/apperr"
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
	if err := m.setReview(ctx, userID, company.ID, "dismissed"); err != nil {
		return dto.CompanyExclusion{}, err
	}
	added, err := m.scoring.ExcludeCompany(ctx, userID, company.Name)
	if err != nil {
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

func (m *Module) ExcludeCompanies(ctx context.Context, userID string, in dto.ExcludeCompaniesInput) (dto.CompaniesExclusion, error) {
	ids, err := distinctCompanyIDs(in.CompanyIDs)
	if err != nil {
		return dto.CompaniesExclusion{}, err
	}
	out := dto.CompaniesExclusion{Companies: make([]dto.CompanyExclusionEntry, 0, len(ids))}
	for _, id := range ids {
		res, err := m.ExcludeCompany(ctx, userID, dto.ExcludeCompanyInput{CompanyID: id})
		if err != nil {
			return dto.CompaniesExclusion{}, err
		}
		out.Companies = append(out.Companies, dto.CompanyExclusionEntry{CompanyID: id, Name: res.Name, Added: res.Added})
	}
	return out, nil
}

func (m *Module) UnexcludeCompanies(ctx context.Context, userID string, in dto.UnexcludeCompaniesInput) (dto.CompaniesExclusion, error) {
	if len(in.Companies) > maxBulkCompanyIDs {
		return dto.CompaniesExclusion{}, apperr.Invalid("too many company IDs")
	}
	seen := make(map[string]bool, len(in.Companies))
	out := dto.CompaniesExclusion{Companies: make([]dto.CompanyExclusionEntry, 0, len(in.Companies))}
	for _, entry := range in.Companies {
		if seen[entry.CompanyID] {
			continue
		}
		seen[entry.CompanyID] = true
		res, err := m.UnexcludeCompany(ctx, userID, dto.UnexcludeCompanyInput(entry))
		if err != nil {
			return dto.CompaniesExclusion{}, err
		}
		out.Companies = append(out.Companies, dto.CompanyExclusionEntry{CompanyID: entry.CompanyID, Name: res.Name})
	}
	return out, nil
}

func distinctCompanyIDs(ids []string) ([]string, error) {
	if len(ids) == 0 {
		return nil, apperr.Invalid("company IDs are required")
	}
	if len(ids) > maxBulkCompanyIDs {
		return nil, apperr.Invalid("too many company IDs")
	}
	out := make([]string, 0, len(ids))
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if id == "" {
			return nil, apperr.Invalid("invalid company ID")
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out, nil
}

func (m *Module) setReview(ctx context.Context, userID, companyID, state string) error {
	_, err := m.store.SetCompanyReviewState(ctx, userID, companyID, state)
	if errors.Is(err, data.ErrNotFound) {
		return nil
	}
	return err
}

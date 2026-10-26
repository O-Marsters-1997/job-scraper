package jobsearch

import (
	"context"
	"math"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/filter"
)

// ListNewCompanies returns the user's Companies awaiting review, each with
// the open Jobs that pass the user's filters and the best Suitability among them.
func (m *Module) ListNewCompanies(ctx context.Context, userID string, q dto.NewCompaniesQuery) (dto.NewCompanyPage, error) {
	pageNumber, err := parsePositiveInt(q.Page, 1, "page")
	if err != nil {
		return dto.NewCompanyPage{}, err
	}
	pageSize, err := parsePositiveInt(q.PageSize, defaultJobPageSize, "page_size")
	if err != nil {
		return dto.NewCompanyPage{}, err
	}
	pageSize = min(pageSize, maxJobPageSize)
	if pageNumber-1 > math.MaxInt32/pageSize {
		return dto.NewCompanyPage{}, apperr.Invalid("page is too large")
	}
	page, err := m.store.ListNewCompanies(ctx, userID, pageSize, (pageNumber-1)*pageSize)
	if err != nil {
		return dto.NewCompanyPage{}, err
	}
	companies := page.Items
	if len(companies) == 0 {
		return page, nil
	}
	cfg, err := m.scoring.SearchConfig(ctx, userID)
	if err != nil {
		return dto.NewCompanyPage{}, err
	}
	ids := make([]string, len(companies))
	for i, c := range companies {
		ids[i] = c.ID
	}
	jobs, err := m.store.ListNewCompanyJobs(ctx, userID, ids)
	if err != nil {
		return dto.NewCompanyPage{}, err
	}
	profiles, err := m.scoring.CompanyProfiles(ctx, userID, ids)
	if err != nil {
		return dto.NewCompanyPage{}, err
	}

	byID := make(map[string]*dto.NewCompany, len(companies))
	for i := range companies {
		c := &companies[i]
		c.Rollup = []dto.CompanyProfileEntry{}
		for _, e := range profiles[c.ID] {
			if e.Known > 0 {
				c.Rollup = append(c.Rollup, e)
			} else {
				c.RollupUnknown++
			}
		}
		for j := range c.Boards {
			b := &c.Boards[j]
			b.URL = detect.BoardURL(b.Source, b.BoardToken)
		}
		byID[c.ID] = c
	}
	for _, job := range jobs {
		c, ok := byID[job.CompanyID]
		if !ok {
			continue
		}
		if _, rejected := filter.Reject(job, cfg); rejected {
			continue
		}
		c.MatchingRoles++
		if s := job.SuitabilityScore; s != nil && (c.BestSuitability == nil || *s > *c.BestSuitability) {
			c.BestSuitability = s
		}
	}
	return page, nil
}

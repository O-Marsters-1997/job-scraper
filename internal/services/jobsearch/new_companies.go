package jobsearch

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/filter"
)

// ListNewCompanies returns the user's Companies awaiting review, each with
// the open Jobs that pass the user's filters and the best Suitability among them.
func (m *Module) ListNewCompanies(ctx context.Context, userID string) ([]dto.NewCompany, error) {
	companies, err := m.store.ListNewCompanies(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(companies) == 0 {
		return companies, nil
	}
	cfg, err := m.scoring.SearchConfig(ctx, userID)
	if err != nil {
		return nil, err
	}
	jobs, err := m.store.ListNewCompanyJobs(ctx, userID)
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(companies))
	for i, c := range companies {
		ids[i] = c.ID
	}
	profiles, err := m.scoring.CompanyProfiles(ctx, userID, ids)
	if err != nil {
		return nil, err
	}

	byID := make(map[string]*dto.NewCompany, len(companies))
	for i := range companies {
		c := &companies[i]
		c.Profile = profiles[c.ID]
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
	return companies, nil
}

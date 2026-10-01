package jobsearch_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
)

func TestListNewCompanies(t *testing.T) {
	st := jobsearchtest.NewFakeStore()
	scoring := jobsearchtest.NewNoopScoring()
	scoring.SeedSearchConfig(dto.SearchConfig{UserID: userID, RequiredTitleKeywords: []string{"go"}})
	deps := jobsearchtest.NewDeps(st)
	deps.Scoring = scoring
	m := jobsearch.Build(deps)

	company := seedAcme(t, st)
	if _, err := st.UpsertCandidateBoard(t.Context(), company.ID, "greenhouse", "acme"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.SetCompanyTracking(t.Context(), userID, company.ID, true, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := st.SetCompanyReviewState(t.Context(), userID, company.ID, "new"); err != nil {
		t.Fatal(err)
	}
	kept := seedCompany(t, st, dto.CompanyUpsert{Slug: "kept", Name: "Kept"})
	if _, err := st.SetCompanyTracking(t.Context(), userID, kept.ID, true, 0); err != nil {
		t.Fatal(err)
	}

	score := func(n int) *int { return &n }
	for i, j := range []dto.Job{
		{Title: "Go Engineer", CompanyID: company.ID, SuitabilityScore: score(60)},
		{Title: "Senior Go Engineer", CompanyID: company.ID, SuitabilityScore: score(85)},
		{Title: "Designer", CompanyID: company.ID, SuitabilityScore: score(99)},
		{Title: "Go Engineer", CompanyID: kept.ID},
	} {
		j.URL = "https://example.com/jobs/" + string(rune('a'+i))
		if _, _, err := st.SaveCanonical(t.Context(), j); err != nil {
			t.Fatal(err)
		}
	}

	got, err := m.ListNewCompanies(t.Context(), userID)
	if err != nil {
		t.Fatalf("ListNewCompanies() err = %v", err)
	}
	if len(got) != 1 || got[0].ID != company.ID {
		t.Fatalf("ListNewCompanies() = %+v, want only the new company", got)
	}
	c := got[0]
	if c.MatchingRoles != 2 || c.BestSuitability == nil || *c.BestSuitability != 85 {
		t.Errorf("ListNewCompanies() = %d roles, best %v, want 2 roles, best 85", c.MatchingRoles, c.BestSuitability)
	}
	if len(c.Boards) != 1 || c.Boards[0].URL == "" {
		t.Errorf("ListNewCompanies() boards = %+v, want one with a URL", c.Boards)
	}
}

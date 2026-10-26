package jobsearch_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
)

func TestListNewCompanies(t *testing.T) {
	st := jobsearchtest.NewFakeStore()
	scoring := jobsearchtest.NewNoopScoring()
	scoring.SeedSearchConfig(dto.SearchConfig{UserID: userID, RequiredTitleKeywords: []string{"go"}})
	profile := []dto.CompanyProfileEntry{{Dimension: dto.DimensionTech, Label: "Go", Yes: 2, Known: 2, Total: 3}}
	deps := jobsearchtest.NewDeps(st)
	deps.Scoring = scoring
	m := jobsearch.Build(deps)

	company := seedAcme(t, st)
	if _, err := st.UpsertCandidateBoard(t.Context(), company.ID, "greenhouse", "acme"); err != nil {
		t.Fatal(err)
	}
	unknown := dto.CompanyProfileEntry{Dimension: dto.DimensionTech, Label: "Rust", Total: 3}
	scoring.SeedProfile(company.ID, append([]dto.CompanyProfileEntry{unknown}, profile...))
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

	page, err := m.ListNewCompanies(t.Context(), userID, dto.NewCompaniesQuery{})
	if err != nil {
		t.Fatalf("ListNewCompanies() err = %v", err)
	}
	got := page.Items
	if page.Total != 1 || len(got) != 1 || got[0].ID != company.ID {
		t.Fatalf("ListNewCompanies() = %+v, want only the new company", got)
	}
	c := got[0]
	if c.MatchingRoles != 2 || c.BestSuitability == nil || *c.BestSuitability != 85 {
		t.Errorf("ListNewCompanies() = %d roles, best %v, want 2 roles, best 85", c.MatchingRoles, c.BestSuitability)
	}
	if diff := cmp.Diff(profile, c.Rollup); diff != "" {
		t.Errorf("ListNewCompanies() profile mismatch (-want +got):\n%s", diff)
	}
	if c.RollupUnknown != 1 {
		t.Errorf("ListNewCompanies() rollup_unknown = %d, want 1", c.RollupUnknown)
	}
	if len(c.Boards) != 1 || c.Boards[0].URL == "" {
		t.Errorf("ListNewCompanies() boards = %+v, want one with a URL", c.Boards)
	}

	t.Run("pages with a total across pages", func(t *testing.T) {
		t.Parallel()
		for _, slug := range []string{"beta", "gamma"} {
			extra := seedCompany(t, st, dto.CompanyUpsert{Slug: slug, Name: slug})
			if _, err := st.SetCompanyTracking(t.Context(), userID, extra.ID, true, 0); err != nil {
				t.Fatal(err)
			}
			if _, err := st.SetCompanyReviewState(t.Context(), userID, extra.ID, "new"); err != nil {
				t.Fatal(err)
			}
		}
		second, err := m.ListNewCompanies(t.Context(), userID, dto.NewCompaniesQuery{Page: "2", PageSize: "2"})
		if err != nil {
			t.Fatalf("ListNewCompanies(page 2) err = %v", err)
		}
		if second.Total != 3 || len(second.Items) != 1 {
			t.Errorf("ListNewCompanies(page 2, size 2) = %d items, total %d, want 1 item, total 3", len(second.Items), second.Total)
		}
		past, err := m.ListNewCompanies(t.Context(), userID, dto.NewCompaniesQuery{Page: "9"})
		if err != nil || past.Items == nil || len(past.Items) != 0 || past.Total != 3 {
			t.Errorf("ListNewCompanies(page 9) = %+v, %v, want empty items, total 3", past, err)
		}
	})

	t.Run("rejects invalid paging", func(t *testing.T) {
		t.Parallel()
		for _, q := range []dto.NewCompaniesQuery{{Page: "0"}, {Page: "x"}, {PageSize: "-1"}, {Page: "2147483647", PageSize: "100"}} {
			if _, err := m.ListNewCompanies(t.Context(), userID, q); !apperr.IsKind(err, apperr.KindInvalid) {
				t.Errorf("ListNewCompanies(%+v) err = %v, want invalid", q, err)
			}
		}
	})
}

package jobsearch_test

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
)

func Example_markAndFilterAgencies() {
	ctx := context.Background()
	st := jobsearchtest.NewFakeStore()
	svc := jobsearch.NewService(st, jobsearchtest.NoopQueue{}, jobsearchtest.NewNoopScoring())
	hays, _ := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "hays", Name: "Hays"})
	_, _ = st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme", ATSSource: "greenhouse", ATSToken: "acme", Kind: dto.KindDirect})

	marked, _ := svc.SetCompanyKind(ctx, "user-1", dto.SetCompanyKindInput{CompanyID: hays.ID, Kind: dto.KindAgency})
	_ = marked.Kind.IsAgency()

	agencies, _ := svc.ListCompanies(ctx, "user-1", dto.CompaniesQuery{Agency: "1"})
	_ = agencies.Items

	_, _ = svc.SetCompanyKind(ctx, "user-1", dto.SetCompanyKindInput{CompanyID: hays.ID, Kind: dto.KindDirect})
}

func Example_classifierSuggestsKind() {
	ctx := context.Background()
	st := jobsearchtest.NewFakeStore()
	m := jobsearch.Build(jobsearchtest.NewDeps(st))
	co, _ := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "talent-co", Name: "Talent Co"})

	landed, _ := m.SuggestCompanyKind(ctx, co.ID, dto.KindLikelyAgency)
	_ = landed

	confirmed := dto.KindAgency.Confirmed()
	_ = confirmed
}

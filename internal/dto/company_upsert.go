package dto

type CompanyUpsert struct {
	Slug, Name, ATSSource, ATSToken, Domain, LinkedInCompanyID string
	Kind                                                       CompanyKind // KindDirect from ATS paths, else empty
}

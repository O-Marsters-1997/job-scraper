package dto

// CompanyUpsert is the input to CompanyProvider.UpsertCompany. Domain and
// LinkedInCompanyID are empty for companies only seen via an ATS source.
type CompanyUpsert struct {
	Slug, Name, ATSSource, ATSToken, Domain, LinkedInCompanyID string
}

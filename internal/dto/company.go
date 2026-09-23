package dto

import "time"

// Company is a shared catalog record of an employer encountered by any scrape.
// ATSSource/ATSToken are empty for companies only seen via discovery sources.
type Company struct {
	ID                string
	Slug              string
	Name              string
	ATSSource         string
	ATSToken          string
	Domain            string
	LinkedInCompanyID string
	LastCrawledAt     *time.Time
	FirstSeenAt       time.Time

	// Per-user read-model fields, populated by ListCompaniesForUser only.
	JobCount             int
	Tracked              bool
	TargetID             string
	CheckIntervalMinutes int
	LastCheckedAt        *time.Time
}

type CompanyTracking struct {
	CompanyID            string
	UserID               string
	Enabled              bool
	CheckIntervalMinutes int
}

package dto

import "time"

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

	JobCount             int
	Tracked              bool
	ReviewState          string
	TargetID             string
	CheckIntervalMinutes int
	LastCheckedAt        *time.Time
}

type CompanyTracking struct {
	CompanyID            string
	UserID               string
	ReviewState          string
	Enabled              bool
	CheckIntervalMinutes int
}

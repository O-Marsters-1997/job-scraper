package dto

import "time"

type Job struct {
	ID                 string
	Title              string
	Location           string
	URL                string
	ApplyURL           string
	CompanySlug        string
	CompanyID          string
	BoardID            string
	ProviderPostingID  string
	ContentFingerprint string
	Source             string
	UpdatedAt          time.Time
	ScrapedAt          time.Time
	Description        string
	SalaryRaw          string
	WorkArrangement    string
	SuitabilityScore   *int
	Breakdown          []ScoreRow
	Wildcard           bool
}

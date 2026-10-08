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
	Band               string
	Grade              string
	Seen               bool
	CompanyFavourite   bool
	Breakdown          []ScoreRow
	Wildcard           bool
	Listings           []JobListing
}

type JobListing struct {
	Source      string    `json:"source"`
	URL         string    `json:"url"`
	FirstSeenAt time.Time `json:"first_seen_at"`
}

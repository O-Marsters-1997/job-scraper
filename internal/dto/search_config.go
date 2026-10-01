package dto

import "time"

type SearchConfig struct {
	ID                    string
	UserID                string
	ExcludedTitleKeywords []string
	ExcludedCompanies     []string
	ExcludedLocations     []string
	RequiredLocations     []string
	RequiredTitleKeywords []string
	NotifyThreshold       int
	CompanyIsNew          bool
	Preferences           Preferences
	UpdatedAt             time.Time
}

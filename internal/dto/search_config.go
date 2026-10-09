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
	MaxJobAgeDays         int
	CompanyIsNew          bool
	CompanyIsFavourite    bool
	Preferences           Preferences
	UpdatedAt             time.Time
}

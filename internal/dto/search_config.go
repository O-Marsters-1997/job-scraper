package dto

import "time"

type SearchConfig struct {
	ID                    string
	UserID                string
	ExcludedTitleKeywords []string
	ExcludedCompanies     []string
	ExcludedLocations     []string
	NotifyThreshold       int
	Preferences           Preferences
	UpdatedAt             time.Time
}

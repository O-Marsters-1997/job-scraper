package dto

import "time"

type ScoringConfigView struct {
	Preferences           Preferences `json:"preferences"`
	ExcludedTitleKeywords []string    `json:"excludedTitleKeywords"`
	ExcludedCompanies     []string    `json:"excludedCompanies"`
	ExcludedLocations     []string    `json:"excludedLocations"`
	RequiredLocations     []string    `json:"requiredLocations"`
	RequiredTitleKeywords []string    `json:"requiredTitleKeywords"`
	NotifyThreshold       int         `json:"notifyThreshold"`
	MaxJobAgeDays         int         `json:"maxJobAgeDays"`
	UpdatedAt             time.Time   `json:"updatedAt"`
	BackfillQueued        int64       `json:"backfillQueued"`
}

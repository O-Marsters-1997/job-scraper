package dto

import "time"

type ScoringConfigView struct {
	Preferences           Preferences `json:"preferences"`
	ExcludedTitleKeywords []string    `json:"excludedTitleKeywords"`
	ExcludedCompanies     []string    `json:"excludedCompanies"`
	ExcludedLocations     []string    `json:"excludedLocations"`
	NotifyThreshold       int         `json:"notifyThreshold"`
	UpdatedAt             time.Time   `json:"updatedAt"`
}

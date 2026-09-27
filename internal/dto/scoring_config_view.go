package dto

import "time"

type ScoringConfigView struct {
	Preferences       Preferences `json:"preferences"`
	ExcludedCompanies []string    `json:"excludedCompanies"`
	ExcludedLocations []string    `json:"excludedLocations"`
	NotifyThreshold   int         `json:"notifyThreshold"`
	UpdatedAt         time.Time   `json:"updatedAt"`
}

package dto

// ScoringConfigView is the wire shape for a user's scoring config, used both
// to decode an update request and to encode the current config.
type ScoringConfigView struct {
	SuitabilityRubric     string   `json:"suitabilityRubric"`
	NotifyThreshold       int      `json:"notifyThreshold"`
	ExcludedTitleKeywords []string `json:"excludedTitleKeywords"`
	ExcludedCompanies     []string `json:"excludedCompanies"`
	ExcludedSeniority     []string `json:"excludedSeniority"`
	ExcludedLocations     []string `json:"excludedLocations"`
}

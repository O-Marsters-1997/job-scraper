package dto

type ScoringConfigView struct {
	SuitabilityRubric     string           `json:"suitabilityRubric"`
	NotifyThreshold       int              `json:"notifyThreshold"`
	ExcludedTitleKeywords []string         `json:"excludedTitleKeywords"`
	ExcludedCompanies     []string         `json:"excludedCompanies"`
	ExcludedSeniority     []string         `json:"excludedSeniority"`
	ExcludedLocations     []string         `json:"excludedLocations"`
	ScoringQuestions      ScoringQuestions `json:"scoringQuestions"`
}

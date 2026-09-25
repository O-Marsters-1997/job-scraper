package dto

import "time"

type SearchConfig struct {
	ID                    string
	UserID                string
	ExcludedTitleKeywords []string
	ExcludedCompanies     []string
	ExcludedSeniority     []string
	ExcludedLocations     []string
	SuitabilityRubric     string
	NotifyThreshold       int
	ScoringQuestions      ScoringQuestions
	UpdatedAt             time.Time
}

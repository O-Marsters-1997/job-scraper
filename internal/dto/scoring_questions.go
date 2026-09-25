package dto

type ScoringCriterion struct {
	Key          string `json:"key"`
	Instructions string `json:"instructions"`
	True         string `json:"true"`
	False        string `json:"false"`
	Required     bool   `json:"required"`
}

type ScoringQuestions struct {
	Profile  string             `json:"profile"`
	Criteria []ScoringCriterion `json:"criteria"`
	Scale    []string           `json:"scale"`
}

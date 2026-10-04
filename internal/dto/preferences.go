package dto

import "time"

// Pick is a user's stance on one bank option; Weight (1-100) is set only on a
// ladder dimension's Picks. Overridden is display-only: a "text" pick whose
// option also has a "manual" pick, which always wins.
type Pick struct {
	OptionID   string `json:"optionId"`
	Stance     string `json:"stance"`
	Weight     int    `json:"weight,omitempty"`
	Source     string `json:"source"`
	Overridden bool   `json:"overridden"`
}

// Preferences is a user's editable scoring input: their picks, a salary
// floor, and the free text extraction derives text picks from.
// PreferenceTextHash is PreferenceText's hash as of the last extraction.
type Preferences struct {
	Picks              []Pick `json:"picks"`
	SalaryFloor        *Money `json:"salaryFloor"`
	PreferenceText     string `json:"preferenceText"`
	PreferenceTextHash string `json:"preferenceTextHash"`
	// Scoring is nil until a fit has run; nil means the Go defaults apply.
	Scoring *ScoringParams `json:"scoring,omitempty"`
}

// BandCuts are the minimum scores for Great, Good and Fair; Poor is below Fair.
type BandCuts struct {
	Great int `json:"great"`
	Good  int `json:"good"`
	Fair  int `json:"fair"`
}

// ScoringParams are a user's fitted overrides of the default dimension
// weights and Band cut-offs. GradeCount is the number of Grades at the last
// fit attempt, accepted or not.
type ScoringParams struct {
	Weights    map[Dimension]float64 `json:"weights,omitempty"`
	Bands      *BandCuts             `json:"bands,omitempty"`
	FittedAt   time.Time             `json:"fittedAt"`
	GradeCount int                   `json:"gradeCount"`
}

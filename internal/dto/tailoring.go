package dto

// CVHeading is a role heading parsed from a base CV Tab. A nil PositionID
// with Confirmed set means the User chose "none"; Confirmed false means the
// mapping was auto-matched or is still open.
type CVHeading struct {
	Text       string  `json:"text"`
	PositionID *string `json:"positionId"`
	Confirmed  bool    `json:"confirmed"`
	SlotCount  int     `json:"slotCount"`
}

// HeadingMapping saves one heading's Position; a nil PositionID is "none".
type HeadingMapping struct {
	HeadingText string  `json:"headingText"`
	PositionID  *string `json:"positionId"`
}

type HeadingMappingsInput struct {
	DocID    string           `json:"-" path:"docId"`
	TabID    string           `json:"-" path:"tabId"`
	Mappings []HeadingMapping `json:"mappings"`
}

// CVTabQuery names one Tab of a base CV Google Doc.
type CVTabQuery struct {
	DocID string `json:"-" path:"docId"`
	TabID string `json:"-" path:"tabId"`
}

// SuggestionsQuery selects the Job to rank Achievements for; DocID and TabID
// optionally name the base CV Tab whose slot counts drive preselection.
type SuggestionsQuery struct {
	JobID string `json:"-" path:"jobId"`
	DocID string `json:"docId"`
	TabID string `json:"tabId"`
}

// Suggestion ranks one Achievement for a Job. Score is P(yes) x confidence.
type Suggestion struct {
	AchievementID string  `json:"achievementId"`
	PositionID    string  `json:"positionId"`
	Text          string  `json:"text"`
	Score         float64 `json:"score"`
	Preselected   bool    `json:"preselected"`
}

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

type SuggestionState string

const (
	SuggestionFit     SuggestionState = "fit"
	SuggestionLow     SuggestionState = "low"
	SuggestionUnclear SuggestionState = "unclear"
)

// Suggestion ranks one Achievement for a Job. Score is the net lean, P(yes) - P(no).
type Suggestion struct {
	AchievementID string          `json:"achievementId"`
	PositionID    string          `json:"positionId"`
	Text          string          `json:"text"`
	Score         float64         `json:"score"`
	State         SuggestionState `json:"state"`
	Preselected   bool            `json:"preselected"`
}

// ExperienceMatchQuery selects the Job to measure the Experience Bank against.
type ExperienceMatchQuery struct {
	JobID string `json:"-" path:"jobId"`
}

// ExperienceMatch is the mean net lean of the Bank's three best Achievements for
// a Job. Score is nil when the Bank is empty.
type ExperienceMatch struct {
	Score *float64 `json:"score"`
}

// ExplainInput names the bullet to explain against a Job.
type ExplainInput struct {
	JobID         string `json:"-" path:"jobId"`
	AchievementID string `json:"-" path:"achievementId"`
}

// Explanation is a model's after-the-fact guess at why a bullet scored low.
type Explanation struct {
	Text string `json:"text"`
}

// SkillSuggestionsQuery selects the Job and the base CV Tab whose Skill Lines
// receive Bank Skill candidates.
type SkillSuggestionsQuery struct {
	JobID string `json:"-" path:"jobId"`
	DocID string `json:"docId"`
	TabID string `json:"tabId"`
}

// SkillItem is a base CV skill and its lean for the Job.
type SkillItem struct {
	Text  string          `json:"text"`
	Score float64         `json:"score"`
	State SuggestionState `json:"state"`
}

// SkillCandidate is a Bank Skill missing from the base CV. Replaces names the
// base item it displaces when Preselected.
type SkillCandidate struct {
	BankSkillID string          `json:"bankSkillId"`
	Name        string          `json:"name"`
	Score       float64         `json:"score"`
	State       SuggestionState `json:"state"`
	Preselected bool            `json:"preselected"`
	Replaces    string          `json:"replaces"`
}

type SkillLineSuggestion struct {
	Label      string           `json:"label"`
	Base       []SkillItem      `json:"base"`
	Candidates []SkillCandidate `json:"candidates"`
}

// SkillSuggestions holds Bank Skill candidates per Skill Line. Unplaced
// candidates have a category that matches no line.
type SkillSuggestions struct {
	Lines    []SkillLineSuggestion `json:"lines"`
	Unplaced []SkillCandidate      `json:"unplaced"`
}

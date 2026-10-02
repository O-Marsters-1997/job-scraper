package dto

import "time"

// ScoreFeedback is one logged note about the scoring: the reason, plus the
// Picks and model in force when it was written.
type ScoreFeedback struct {
	ID           string                `json:"id"`
	Kind         string                `json:"kind"`
	Direction    *string               `json:"direction,omitempty"`
	JobID        *string               `json:"jobId,omitempty"`
	Reason       string                `json:"reason"`
	Picks        []Pick                `json:"picks"`
	Model        string                `json:"model"`
	Snapshot     ScoreFeedbackSnapshot `json:"snapshot"`
	CreatedAt    time.Time             `json:"createdAt"`
	PicksChanged bool                  `json:"picksChanged"`
	ModelChanged bool                  `json:"modelChanged"`
}

// ScoreFeedbackSnapshot is what a kind froze beside its reason: a Job entry
// freezes the score evidence, an Overall entry nothing.
type ScoreFeedbackSnapshot struct {
	Score              *int             `json:"score,omitempty"`
	Breakdown          []ScoreRow       `json:"breakdown,omitempty"`
	ScoreFingerprint   string           `json:"scoreFingerprint,omitempty"`
	ScoreModel         string           `json:"scoreModel,omitempty"`
	ContentFingerprint string           `json:"contentFingerprint,omitempty"`
	Options            []FeedbackOption `json:"options,omitempty"`
	JevState           *JevState        `json:"jevState,omitempty"`
}

// FeedbackOption is one picked Option as the score saw it: Jev's cached
// probabilities and how they resolved. Known is false when no answer was cached.
type FeedbackOption struct {
	OptionID   string  `json:"optionId"`
	Label      string  `json:"label"`
	Question   string  `json:"question"`
	Stance     string  `json:"stance"`
	Resolved   string  `json:"resolved"`
	PYes       float64 `json:"pYes"`
	PNo        float64 `json:"pNo"`
	PNotStated float64 `json:"pNotStated"`
	Confidence float64 `json:"confidence"`
	Known      bool    `json:"known"`
}

// JevState is the job as Jev is sent it: HTML stripped and the description
// truncated.
type JevState struct {
	Title           string `json:"title"`
	Company         string `json:"company"`
	Location        string `json:"location"`
	WorkArrangement string `json:"work_arrangement"`
	SalaryRaw       string `json:"salary_raw"`
	Description     string `json:"description"`
}

// JobScoreEvidence is the stored score a Job entry freezes.
type JobScoreEvidence struct {
	Score       int
	Breakdown   []ScoreRow
	Fingerprint string
	Model       string
}

type JobFeedbackInput struct {
	JobID     string `json:"jobId"`
	Direction string `json:"direction"`
	Reason    string `json:"reason"`
}

// ScoreFeedbackQuery filters and pages the log. Kind is empty for all kinds;
// Page is 1-based and defaults to 1.
type ScoreFeedbackQuery struct {
	Kind     string `json:"kind"`
	Outdated string `json:"outdated"`
	Page     string `json:"page"`
}

// ScoreFeedbackFilter selects entries for the store: Model is the live model
// that drift is measured against, and outdated entries are listed only when
// IncludeOutdated is set.
type ScoreFeedbackFilter struct {
	Kind            string
	Model           string
	IncludeOutdated bool
}

// ScoreFeedbackPage is one page of the log. Total counts what the query
// matches; CurrentCount and OutdatedCount split the matching kind's entries.
type ScoreFeedbackPage struct {
	Entries       []ScoreFeedback `json:"entries"`
	Total         int             `json:"total"`
	CurrentCount  int             `json:"currentCount"`
	OutdatedCount int             `json:"outdatedCount"`
}

type OverallFeedbackInput struct {
	Reason string `json:"reason"`
}

package dto

import "time"

// ScoreFeedback is one logged note about the scoring: the reason, plus the
// Picks and model in force when it was written.
type ScoreFeedback struct {
	ID        string                `json:"id"`
	Kind      string                `json:"kind"`
	Reason    string                `json:"reason"`
	Picks     []Pick                `json:"picks"`
	Model     string                `json:"model"`
	Snapshot  ScoreFeedbackSnapshot `json:"snapshot"`
	CreatedAt time.Time             `json:"createdAt"`
}

// ScoreFeedbackSnapshot is what a kind froze beside its reason; an Overall
// entry freezes nothing.
type ScoreFeedbackSnapshot struct{}

// ScoreFeedbackQuery filters and pages the log. Kind is empty for all kinds;
// Page is 1-based and defaults to 1.
type ScoreFeedbackQuery struct {
	Kind string `json:"kind"`
	Page string `json:"page"`
}

type ScoreFeedbackPage struct {
	Entries []ScoreFeedback `json:"entries"`
	Total   int             `json:"total"`
}

type OverallFeedbackInput struct {
	Reason string `json:"reason"`
}

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

type OverallFeedbackInput struct {
	Reason string `json:"reason"`
}

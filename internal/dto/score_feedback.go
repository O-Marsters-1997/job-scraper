package dto

import "time"

// ScoreFeedback is one logged note about the scoring: the reason, plus the
// Picks and model in force when it was written.
type ScoreFeedback struct {
	ID        string                `json:"id"`
	Kind      string                `json:"kind"`
	Direction *string               `json:"direction,omitempty"`
	JobID     *string               `json:"jobId,omitempty"`
	Reason    string                `json:"reason"`
	Picks     []Pick                `json:"picks"`
	Model     string                `json:"model"`
	Snapshot  ScoreFeedbackSnapshot `json:"snapshot"`
	CreatedAt time.Time             `json:"createdAt"`
}

// ScoreFeedbackSnapshot is what a kind froze beside its reason: a Job entry
// freezes the score evidence, an Overall entry nothing.
type ScoreFeedbackSnapshot struct {
	Score              *int              `json:"score,omitempty"`
	Breakdown          []ScoreRow        `json:"breakdown,omitempty"`
	ScoreFingerprint   string            `json:"scoreFingerprint,omitempty"`
	ScoreModel         string            `json:"scoreModel,omitempty"`
	ContentFingerprint string            `json:"contentFingerprint,omitempty"`
	Options            []FeedbackOption  `json:"options,omitempty"`
	JevState           *JevState         `json:"jevState,omitempty"`
	Filters            map[string]string `json:"filters,omitempty"`
	Ranking            []RankedJob       `json:"ranking,omitempty"`
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

// RankedJob is one row of a Collection entry's ranking as the user saw it.
// Score is nil when the Job had no score.
type RankedJob struct {
	Rank    int      `json:"rank"`
	JobID   string   `json:"jobId"`
	Title   string   `json:"title"`
	Company string   `json:"company"`
	Score   *int     `json:"score"`
	Effects []string `json:"effects"`
}

// CollectionJobScore is a Job's stored score as read for a Collection entry.
type CollectionJobScore struct {
	JobID     string
	Title     string
	Company   string
	Score     *int
	Breakdown []ScoreRow
}

type CollectionFeedbackInput struct {
	JobIDs  []string          `json:"jobIds"`
	Filters map[string]string `json:"filters"`
	Reason  string            `json:"reason"`
}

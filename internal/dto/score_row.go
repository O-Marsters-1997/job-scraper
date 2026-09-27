package dto

// ScoreRow is one pick's contribution to a JobScore: what the pick was,
// what the job's cached answer resolved to, and how that told on the score.
type ScoreRow struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	Stance     string `json:"stance"`
	Resolved   string `json:"resolved"` // "yes" | "no" | "unknown"
	Effect     string `json:"effect"`   // "meets" | "misses" | "unknown" | "neutral"
	Overridden bool   `json:"overridden"`
}

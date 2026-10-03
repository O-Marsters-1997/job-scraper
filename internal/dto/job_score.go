package dto

// JobScore is one user's computed Suitability for one job.
type JobScore struct {
	JobID    string     `json:"jobId"`
	UserID   string     `json:"-"`
	Score    int        `json:"score"`
	Band     string     `json:"band"`
	Rows     []ScoreRow `json:"rows"`
	Unknowns int        `json:"unknowns"`
	Cost     float64    `json:"-"`
}

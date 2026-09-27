package dto

// JobScore is one user's computed Suitability for one job.
type JobScore struct {
	JobID    string
	UserID   string
	Score    int
	Rows     []ScoreRow
	Unknowns int
	Cost     float64
	Hidden   bool
}

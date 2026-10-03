package dto

import "time"

// Grade is a user's current verdict on one Job: "great", "ok" or "no". A "no"
// is also what Dismiss writes.
type Grade struct {
	JobID        string    `json:"jobId"`
	Grade        string    `json:"grade"`
	Reasons      []string  `json:"reasons"`
	ScoreAtGrade *int      `json:"scoreAtGrade,omitempty"`
	ScoreModel   string    `json:"scoreModel,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type GradeInput struct {
	JobID   string   `json:"-" path:"id"`
	Grade   string   `json:"grade"`
	Reasons []string `json:"reasons"`
}

// ImpliedLabel is a Job the User engaged with beyond a Grade: Source is
// "application" or "kept_cv".
type ImpliedLabel struct {
	JobID  string
	Source string
}

package dto

import "time"

type JobNote struct {
	ID        string
	JobID     string
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateJobNoteInput struct {
	JobID string `json:"-" path:"id"`
	Body  string `json:"body"`
}

type UpdateJobNoteInput struct {
	ID   string `json:"-" path:"id"`
	Body string `json:"body"`
}

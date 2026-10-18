package dto

import "time"

type NoteQuote struct {
	Exact  string `json:"exact"`
	Prefix string `json:"prefix"`
	Suffix string `json:"suffix"`
}

type JobNote struct {
	ID        string
	JobID     string
	Body      string
	Quote     *NoteQuote
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateJobNoteInput struct {
	JobID string     `json:"-" path:"id"`
	Body  string     `json:"body"`
	Quote *NoteQuote `json:"quote"`
}

type UpdateJobNoteInput struct {
	ID   string `json:"-" path:"id"`
	Body string `json:"body"`
}

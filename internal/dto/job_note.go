package dto

type JobNote struct {
	JobID string
	Body  string
}

type PutJobNoteInput struct {
	JobID string `json:"-" path:"id"`
	Body  string `json:"body"`
}

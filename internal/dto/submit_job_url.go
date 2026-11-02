package dto

type SubmissionStatus string

const (
	SubmissionHeld   SubmissionStatus = "held"
	SubmissionQueued SubmissionStatus = "queued"
)

type SubmitJobURLInput struct {
	URL string `json:"url"`
}

type SubmitJobURLResult struct {
	Status SubmissionStatus `json:"status"`
	JobID  string           `json:"jobId,omitempty"`
}

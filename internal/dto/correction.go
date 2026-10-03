package dto

// CorrectionInput is a user's claim that a job's answer to one option is
// Value ("yes" or "no"), whatever the model said.
type CorrectionInput struct {
	JobID    string `json:"-" path:"id"`
	OptionID string `json:"-" path:"optionId"`
	Value    string `json:"value"`
}

// RevertCorrectionInput removes a user's Correction of one option on a job.
type RevertCorrectionInput struct {
	JobID    string `json:"-" path:"id"`
	OptionID string `json:"-" path:"optionId"`
}

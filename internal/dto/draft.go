package dto

import "encoding/json"

// DraftInput asks for a Tailored CV Draft of one Job from one base CV Tab,
// citing the Achievements the User confirmed.
type DraftInput struct {
	JobID          string   `json:"jobId"`
	DocID          string   `json:"docId"`
	TabID          string   `json:"tabId"`
	AchievementIDs []string `json:"achievementIds"`
}

// MaxDraftAttempts is how many times a Draft is claimed before it fails for
// good.
const MaxDraftAttempts = 5

type DraftRef struct {
	ID string `json:"id"`
}

// Draft is a Tailored CV's generation state. Status is pending, running,
// ready or failed; DraftDocURL is set once the Doc exists.
type Draft struct {
	ID          string  `json:"id"`
	JobID       string  `json:"jobId"`
	Status      string  `json:"status"`
	DraftDocURL *string `json:"draftDocUrl"`
	LastError   string  `json:"lastError"`
	DraftDocID  string  `json:"-"`
}

// DraftClaim is a Draft leased to the generator with the Job facts it
// needs.
type DraftClaim struct {
	ID             string
	UserID         string
	JobID          string
	DocID          string
	TabID          string
	AchievementIDs []string
	Attempts       int
	DraftDocID     string
	JobDescription string
	JobFingerprint string
}

// DraftResult is what a finished generation records on its Draft.
type DraftResult struct {
	EditSet        json.RawMessage
	RawOutput      string
	Model          string
	PromptVersion  string
	JobFingerprint string
	Cost           float64
	DraftDocID     string
}

// DraftFailure records why a claimed Draft attempt failed. Terminal skips
// the remaining retries; ClearDoc drops the Draft's Doc id once the Doc is
// deleted.
type DraftFailure struct {
	Reason   string
	Terminal bool
	ClearDoc bool
}

type DraftQuery struct {
	ID string `json:"-" path:"id"`
}

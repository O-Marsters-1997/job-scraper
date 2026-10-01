package dto

import (
	"encoding/json"
	"time"
)

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

// DraftFinding is one check result recorded on a Draft. Severity is block,
// warn or info.
type DraftFinding struct {
	Check    string   `json:"check"`
	Severity string   `json:"severity"`
	SlotID   string   `json:"slotId,omitempty"`
	Message  string   `json:"message"`
	Score    *float64 `json:"score,omitempty"`
}

// Draft is a Tailored CV's generation state. Status is pending, running,
// ready or failed; DraftDocURL is set while the Doc exists. Outcome is nil
// until the User keeps or discards a ready Draft. Provenance is set by
// GetDraft only.
type Draft struct {
	ID          string           `json:"id"`
	JobID       string           `json:"jobId"`
	Status      string           `json:"status"`
	Outcome     *string          `json:"outcome"`
	DraftDocURL *string          `json:"draftDocUrl"`
	LastError   string           `json:"lastError"`
	CreatedAt   time.Time        `json:"createdAt"`
	Findings    []DraftFinding   `json:"findings"`
	Provenance  *DraftProvenance `json:"provenance"`
	DraftDocID  string           `json:"-"`
	EditSet     json.RawMessage  `json:"-"`
	Content     *DraftContent    `json:"content"`
	Base        *DraftContent    `json:"base"`
	BaseContent json.RawMessage  `json:"-"`
}

// DraftContent is the editable part of a Draft: the Profile, Skills and each
// Position's bullets in CV order. Base holds the base CV Tab's content as it
// was at generation, with no AchievementIDs.
type DraftContent struct {
	Profile   *string         `json:"profile"`
	Skills    []string        `json:"skills"`
	Positions []DraftPosition `json:"positions"`
}

type DraftPosition struct {
	PositionID string        `json:"positionId"`
	Bullets    []DraftBullet `json:"bullets"`
}

type DraftBullet struct {
	Text           string   `json:"text"`
	AchievementIDs []string `json:"achievementIds"`
}

const (
	OutcomeKept      = "kept"
	OutcomeDiscarded = "discarded"
)

// DraftProvenance shows where each bullet of a Draft came from.
type DraftProvenance struct {
	Positions []ProvenancePosition `json:"positions"`
}

type ProvenancePosition struct {
	PositionID string             `json:"positionId"`
	Employer   string             `json:"employer"`
	Title      string             `json:"title"`
	Bullets    []ProvenanceBullet `json:"bullets"`
}

// ProvenanceBullet is one rewritten bullet, its cited Achievements, and its
// text split so words absent from those Achievements are marked Novel.
type ProvenanceBullet struct {
	Segments     []TextSegment `json:"segments"`
	Achievements []Achievement `json:"achievements"`
}

type TextSegment struct {
	Text  string `json:"text"`
	Novel bool   `json:"novel"`
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
	BaseContent    json.RawMessage
	RawOutput      string
	Model          string
	PromptVersion  string
	JobFingerprint string
	Cost           float64
	DraftDocID     string
	Findings       []DraftFinding
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

// DraftJobQuery selects the Drafts of one Job.
type DraftJobQuery struct {
	JobID string `json:"-" path:"jobId"`
}

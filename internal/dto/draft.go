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
// keeping, ready or failed; DraftDocURL is set while the Doc exists. Outcome
// is nil until the User keeps or discards a ready Draft. Provenance is set
// by GetDraft only.
type Draft struct {
	ID          string           `json:"id"`
	JobID       string           `json:"jobId"`
	Status      string           `json:"status"`
	Outcome     *string          `json:"outcome"`
	KeptAs      string           `json:"keptAs"`
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

	BaseDocID      string   `json:"-"`
	BaseTabID      string   `json:"-"`
	AchievementIDs []string `json:"-"`
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
	Profile   *ProvenanceProfile   `json:"profile"`
}

// ProvenanceProfile is the Profile text split so words absent from the
// Achievement bank are marked Novel.
type ProvenanceProfile struct {
	SlotID   string        `json:"slotId"`
	Segments []TextSegment `json:"segments"`
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
	SlotID       string        `json:"slotId"`
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
	Keeping        bool
	JobTitle       string
	CompanyName    string
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

// SlotEdit is the User's new text for one bullet slot of a Draft's Doc.
type SlotEdit struct {
	SlotID string `json:"slotId"`
	Text   string `json:"text"`
}

// DraftSlotsInput carries the bullets the User changed on a Draft.
type DraftSlotsInput struct {
	ID    string     `json:"-" path:"id"`
	Slots []SlotEdit `json:"slots"`
}

// DraftLayout is a Draft Doc's single-column page geometry and styled
// paragraphs, in pt, for rendering at true size.
type DraftLayout struct {
	Page     LayoutPage    `json:"page"`
	Blocks   []LayoutBlock `json:"blocks"`
	Warnings []string      `json:"warnings"`
}

type LayoutPage struct {
	Width        float64 `json:"width"`
	Height       float64 `json:"height"`
	MarginTop    float64 `json:"marginTop"`
	MarginBottom float64 `json:"marginBottom"`
	MarginLeft   float64 `json:"marginLeft"`
	MarginRight  float64 `json:"marginRight"`
}

// LayoutBlock is one paragraph. SlotID is "", "profile" or the base slot ID;
// Section is "skills" inside the read-only Skills range.
type LayoutBlock struct {
	SlotID          string        `json:"slotId"`
	Section         string        `json:"section"`
	Align           string        `json:"align"`
	LineSpacing     float64       `json:"lineSpacing"`
	SpaceAbove      float64       `json:"spaceAbove"`
	SpaceBelow      float64       `json:"spaceBelow"`
	IndentStart     float64       `json:"indentStart"`
	IndentFirstLine float64       `json:"indentFirstLine"`
	BorderTop       *LayoutBorder `json:"borderTop"`
	BorderBottom    *LayoutBorder `json:"borderBottom"`
	TabStops        []LayoutTab   `json:"tabStops"`
	Bullet          *LayoutBullet `json:"bullet"`
	Runs            []LayoutRun   `json:"runs"`
}

type LayoutBorder struct {
	Width   float64 `json:"width"`
	Color   string  `json:"color"`
	Padding float64 `json:"padding"`
	Dash    string  `json:"dash"`
}

type LayoutTab struct {
	Offset    float64 `json:"offset"`
	Alignment string  `json:"alignment"`
}

type LayoutBullet struct {
	Glyph string  `json:"glyph"`
	Level int     `json:"level"`
	Size  float64 `json:"size"`
}

// LayoutRun is a styled text run. An empty paragraph's single "\n" run
// carries its size.
type LayoutRun struct {
	Text      string  `json:"text"`
	Font      string  `json:"font"`
	Size      float64 `json:"size"`
	Bold      bool    `json:"bold"`
	Italic    bool    `json:"italic"`
	Underline bool    `json:"underline"`
	Color     string  `json:"color"`
	Link      string  `json:"link"`
}

// SuggestInput asks for one inline edit of a slot. Text is the client's
// current, possibly unsaved, text; MaxChars applies to the fit action.
type SuggestInput struct {
	ID       string `json:"-" path:"id"`
	SlotID   string `json:"-" path:"slotId"`
	Action   string `json:"action"`
	Prompt   string `json:"prompt"`
	Text     string `json:"text"`
	MaxChars int    `json:"maxChars"`
}

// SuggestDone is the final event of a suggestion stream.
type SuggestDone struct {
	Text     string         `json:"text"`
	Findings []DraftFinding `json:"findings"`
}

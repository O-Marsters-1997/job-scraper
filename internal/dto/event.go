package dto

import (
	"encoding/json"
	"time"
)

// EventInput is a frontend event: Type is one of the client-reportable event
// types, SubjectID the job it concerns, Reason an optional dismiss reason.
type EventInput struct {
	Type      string `json:"type"`
	SubjectID string `json:"subject_id"`
	Reason    string `json:"reason"`
	Undone    bool   `json:"undone"`
}

type ImpressionSurface string

const (
	SurfaceTriage ImpressionSurface = "triage"
	SurfaceList   ImpressionSurface = "list"
)

type ImpressionItem struct {
	JobID    string `json:"job_id"`
	Position int    `json:"position"`
}

type ImpressionBatchInput struct {
	Surface ImpressionSurface `json:"surface"`
	Sort    JobSort           `json:"sort"`
	Items   []ImpressionItem  `json:"items"`
}

// Event is a stored events row; Props is the per-type JSON payload.
type Event struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	SubjectID string          `json:"subject_id,omitempty"`
	Client    string          `json:"client"`
	Props     json.RawMessage `json:"props"`
	CreatedAt time.Time       `json:"created_at"`
}

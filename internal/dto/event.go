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
}

// Event is a stored events row; Props is the per-type JSON payload.
type Event struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	SubjectID string          `json:"subject_id,omitempty"`
	Props     json.RawMessage `json:"props"`
	CreatedAt time.Time       `json:"created_at"`
}

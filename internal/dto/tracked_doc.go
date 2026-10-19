package dto

import "time"

type TrackedDocKind string

const (
	TrackedDocCV     TrackedDocKind = "cv"
	TrackedDocLetter TrackedDocKind = "letter"
)

type TrackedDoc struct {
	ID      string
	UserID  string
	DocID   string
	Kind    TrackedDocKind
	AddedAt time.Time
}

type AddTrackedDocInput struct {
	UserID string
	DocID  string
	Kind   TrackedDocKind
}

type TrackedDocInput struct {
	URL  string         `json:"url"`
	Kind TrackedDocKind `json:"kind"`
}

type TrackedDocKindQuery struct {
	Kind TrackedDocKind `json:"kind"`
}

type TabVisibilityInput struct {
	DocID string `json:"-" path:"docId"`
	TabID string `json:"-" path:"tabId"`
}

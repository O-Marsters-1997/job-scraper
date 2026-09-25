package dto

import "time"

// TrackedDoc represents a Google Doc being tracked as a CV template source.
type TrackedDoc struct {
	ID      string
	UserID  string
	DocID   string
	AddedAt time.Time
}

type AddTrackedDocInput struct {
	UserID string
	DocID  string
}

// TrackedDocInput is the wire body for POST /tracked-docs.
type TrackedDocInput struct {
	URL string `json:"url"`
}

// TabVisibilityInput is the path-only body for the tab hide/show routes.
type TabVisibilityInput struct {
	DocID string `json:"-" path:"docId"`
	TabID string `json:"-" path:"tabId"`
}

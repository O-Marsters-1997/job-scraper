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

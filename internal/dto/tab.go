package dto

import "time"

type Tab struct {
	ID           string
	TrackedDocID string
	TabID        string
	Title        string
	Visible      bool
	IsDefault    bool
	CreatedAt    time.Time
}

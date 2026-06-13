package dto

import "time"

type Tab struct {
	ID           string
	TrackedDocID string
	TabID        string
	Title        string
	Visible      bool
	CreatedAt    time.Time
}

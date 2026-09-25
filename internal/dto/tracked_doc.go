package dto

import "time"

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

type TrackedDocInput struct {
	URL string `json:"url"`
}

type TabVisibilityInput struct {
	DocID string `json:"-" path:"docId"`
	TabID string `json:"-" path:"tabId"`
}

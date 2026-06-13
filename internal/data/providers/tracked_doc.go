package providers

import (
	"context"
	"errors"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// ErrTrackedDocNotFound is returned when no tracked doc exists for the given user and docID.
var ErrTrackedDocNotFound = errors.New("tracked doc not found")

// TrackedDocProvider persists tracked Google Doc references.
type TrackedDocProvider interface {
	AddTrackedDoc(ctx context.Context, input dto.AddTrackedDocInput) error
	RemoveTrackedDoc(ctx context.Context, userID, docID string) error
	ListTrackedDocs(ctx context.Context, userID string) ([]dto.TrackedDoc, error)
}

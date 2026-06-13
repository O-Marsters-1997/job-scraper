package providers

import (
	"context"
	"errors"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

var ErrTrackedDocNotFound = errors.New("tracked doc not found")

type TrackedDocProvider interface {
	AddTrackedDoc(ctx context.Context, input dto.AddTrackedDocInput) error
	RemoveTrackedDoc(ctx context.Context, userID, docID string) error
	ListTrackedDocs(ctx context.Context, userID string) ([]dto.TrackedDoc, error)
}

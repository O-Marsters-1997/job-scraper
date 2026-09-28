package cvtemplates

import (
	"context"
	"fmt"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/docref"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

var (
	ErrInvalidDoc      = apperr.Invalid("invalid Google Docs URL or ID")
	ErrInaccessibleDoc = apperr.NotFound("cannot access document")
)

// AddDoc tracks the Google Doc at in.URL, after confirming userID can
// access it.
func (s *Service) AddDoc(ctx context.Context, userID string, in dto.TrackedDocInput) (struct{}, error) {
	docID, err := docref.ParseDocID(in.URL)
	if err != nil {
		return struct{}{}, fmt.Errorf("%w: %w", ErrInvalidDoc, err)
	}
	if _, err := s.gc.FileMeta(ctx, userID, docID); err != nil {
		return struct{}{}, fmt.Errorf("%w: %w", ErrInaccessibleDoc, err)
	}
	return struct{}{}, s.store.AddTrackedDoc(ctx, dto.AddTrackedDocInput{UserID: userID, DocID: docID})
}

func (s *Service) RemoveDoc(ctx context.Context, userID, docID string) error {
	return s.store.RemoveTrackedDoc(ctx, userID, docID)
}

func (s *Service) HideTab(ctx context.Context, userID string, in dto.TabVisibilityInput) (struct{}, error) {
	return struct{}{}, s.store.HideTab(ctx, userID, in.DocID, in.TabID)
}

func (s *Service) ShowTab(ctx context.Context, userID string, in dto.TabVisibilityInput) (struct{}, error) {
	return struct{}{}, s.store.ShowTab(ctx, userID, in.DocID, in.TabID)
}

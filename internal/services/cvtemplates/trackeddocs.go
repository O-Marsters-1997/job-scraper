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
	ErrInvalidKind     = apperr.Invalid("invalid tracked doc kind")
)

func normalizeKind(kind dto.TrackedDocKind) (dto.TrackedDocKind, error) {
	switch kind {
	case "":
		return dto.TrackedDocCV, nil
	case dto.TrackedDocCV, dto.TrackedDocLetter:
		return kind, nil
	}
	return "", ErrInvalidKind
}

// AddDoc tracks the Google Doc at in.URL, after confirming userID can
// access it.
func (s *Service) AddDoc(ctx context.Context, userID string, in dto.TrackedDocInput) (struct{}, error) {
	kind, err := normalizeKind(in.Kind)
	if err != nil {
		return struct{}{}, err
	}
	docID, err := docref.ParseDocID(in.URL)
	if err != nil {
		return struct{}{}, fmt.Errorf("%w: %w", ErrInvalidDoc, err)
	}
	if _, err := s.gc.FileMeta(ctx, userID, docID); err != nil {
		return struct{}{}, fmt.Errorf("%w: %w", ErrInaccessibleDoc, err)
	}
	return struct{}{}, s.store.AddTrackedDoc(ctx, dto.AddTrackedDocInput{UserID: userID, DocID: docID, Kind: kind})
}

func (s *Service) HideTab(ctx context.Context, userID string, in dto.TabVisibilityInput) (struct{}, error) {
	return struct{}{}, s.store.SetTabVisible(ctx, userID, in.DocID, in.TabID, false)
}

func (s *Service) SetDefaultTab(ctx context.Context, userID string, in dto.TabVisibilityInput) (struct{}, error) {
	return struct{}{}, s.store.SetTabDefault(ctx, userID, in.DocID, in.TabID)
}

func (s *Service) ShowTab(ctx context.Context, userID string, in dto.TabVisibilityInput) (struct{}, error) {
	return struct{}{}, s.store.SetTabVisible(ctx, userID, in.DocID, in.TabID, true)
}

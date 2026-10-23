package cvtailor

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func (s *Service) CreateDraft(ctx context.Context, userID string, in dto.DraftInput) (dto.DraftRef, error) {
	if in.JobID == "" || in.DocID == "" || in.TabID == "" {
		return dto.DraftRef{}, apperr.Invalid("jobId, docId and tabId are required")
	}
	switch in.Mode {
	case dto.DraftModeAsIs, dto.DraftModeGenerate:
	case "":
		in.Mode = dto.DraftModeGenerate
	default:
		return dto.DraftRef{}, apperr.Invalid("unknown mode")
	}
	draft, err := s.store.CreateDraft(ctx, userID, in)
	if err != nil {
		return dto.DraftRef{}, err
	}
	return dto.DraftRef{ID: draft.ID}, nil
}

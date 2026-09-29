package cvtailor

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

// CreateDraft queues a pending Draft. Every cited Achievement must belong
// to userID and sit under a Position the User has mapped a heading of the
// base CV Tab to.
func (s *Service) CreateDraft(ctx context.Context, userID string, in dto.DraftInput) (dto.DraftRef, error) {
	if in.JobID == "" || in.DocID == "" || in.TabID == "" {
		return dto.DraftRef{}, apperr.Invalid("jobId, docId and tabId are required")
	}
	if len(in.AchievementIDs) == 0 {
		return dto.DraftRef{}, apperr.Invalid("choose at least one achievement")
	}
	if hasDuplicates(in.AchievementIDs) {
		return dto.DraftRef{}, apperr.Invalid("achievementIds must not repeat")
	}
	positions, err := s.store.ListPositions(ctx, userID)
	if err != nil {
		return dto.DraftRef{}, err
	}
	mappings, err := s.store.ListHeadingMappings(ctx, userID, in.DocID, in.TabID)
	if err != nil {
		return dto.DraftRef{}, err
	}
	mapped := make(map[string]bool, len(mappings))
	for _, m := range mappings {
		if m.PositionID != nil {
			mapped[*m.PositionID] = true
		}
	}
	positionOf := map[string]string{}
	for _, p := range positions {
		for _, a := range p.Achievements {
			positionOf[a.ID] = p.ID
		}
	}
	for _, id := range in.AchievementIDs {
		pid, ok := positionOf[id]
		if !ok {
			return dto.DraftRef{}, apperr.Invalid("unknown achievement")
		}
		if !mapped[pid] {
			return dto.DraftRef{}, apperr.Invalid("map the CV's headings to your positions first")
		}
	}
	draft, err := s.store.CreateDraft(ctx, userID, in)
	if err != nil {
		return dto.DraftRef{}, err
	}
	return dto.DraftRef{ID: draft.ID}, nil
}

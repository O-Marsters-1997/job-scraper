package cvtailor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
)

// SaveDraftSlots writes the User's bullet edits into the Draft's Doc and
// re-runs the checks. Findings never block a save.
func (s *Service) SaveDraftSlots(ctx context.Context, userID string, in dto.DraftSlotsInput) (dto.Draft, error) {
	draft, err := s.store.GetDraft(ctx, userID, in.ID)
	if err != nil {
		return dto.Draft{}, err
	}
	if draft.Status != statusReady || draft.DraftDocID == "" {
		return dto.Draft{}, apperr.Conflict("only a ready draft can be edited")
	}
	if outcome(draft) == dto.OutcomeDiscarded {
		return dto.Draft{}, apperr.Conflict("this draft was discarded")
	}
	var edits cvedit.EditSet
	if err := json.Unmarshal(draft.EditSet, &edits); err != nil {
		return dto.Draft{}, fmt.Errorf("decode edit set: %w", err)
	}

	claim := dto.DraftClaim{ID: draft.ID, UserID: userID, DocID: draft.BaseDocID, TabID: draft.BaseTabID, AchievementIDs: draft.AchievementIDs}
	basePlan, err := s.plan(ctx, claim, claim.DocID)
	if err != nil {
		return dto.Draft{}, err
	}
	docPlan, err := s.plan(ctx, claim, draft.DraftDocID)
	if err != nil {
		return dto.Draft{}, err
	}
	edits, err = basePlan.overlay(edits, in.Slots)
	if err != nil {
		return dto.Draft{}, err
	}
	if err := s.applyEdits(ctx, userID, draft.DraftDocID, docPlan, cvedit.EditSet{Positions: edits.Positions}); err != nil {
		return dto.Draft{}, err
	}

	basePages, draftPages, draftPDF, err := s.pageCounts(ctx, claim, draft.DraftDocID)
	if err != nil {
		return dto.Draft{}, err
	}
	editSet, err := json.Marshal(edits)
	if err != nil {
		return dto.Draft{}, fmt.Errorf("marshal edit set: %w", err)
	}
	checked := basePlan.draft(edits, basePages, draftPages)
	checked.Parse = parseInput(ctx, draftPDF, basePlan.structure.Headings)
	findings := toDraftFindings(checks.Run(checked))
	if err := s.store.SetDraftEdits(ctx, userID, draft.ID, editSet, findings); err != nil {
		return dto.Draft{}, err
	}
	return s.GetDraft(ctx, userID, dto.DraftQuery{ID: draft.ID})
}

func (pl plan) overlay(edits cvedit.EditSet, slots []dto.SlotEdit) (cvedit.EditSet, error) {
	texts := make(map[string]string, len(slots))
	for _, e := range slots {
		text := strings.TrimSpace(e.Text)
		if text == "" || strings.ContainsAny(text, "\r\n") {
			return edits, apperr.Invalid("a bullet must be one non-empty line")
		}
		texts[e.SlotID] = text
	}
	slotIDs := pl.slotIDs()
	for _, pe := range edits.Positions {
		if len(pe.Bullets) > len(slotIDs[pe.PositionID]) {
			return edits, apperr.Conflict("the CV's headings changed since this draft was made")
		}
		for i := range pe.Bullets {
			id := slotIDs[pe.PositionID][i]
			if text, ok := texts[id]; ok && text != pe.Bullets[i].Text {
				pe.Bullets[i].Text, pe.Bullets[i].Keep = text, false
			}
			delete(texts, id)
		}
	}
	if len(texts) > 0 {
		return edits, apperr.Invalid("unknown slot")
	}
	return edits, nil
}

package cvtailor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/docparse"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/tabcopy"
)

const skillsSection = "skills"

// DraftLayout resolves the Draft Doc's page for rendering at true size, with
// each editable paragraph tagged by the slot ID the Draft's provenance uses.
func (s *Service) DraftLayout(ctx context.Context, userID string, q dto.DraftQuery) (dto.DraftLayout, error) {
	draft, err := s.store.GetDraft(ctx, userID, q.ID)
	if err != nil {
		return dto.DraftLayout{}, err
	}
	if (draft.Status != statusReady && draft.Status != statusKeeping) || draft.DraftDocID == "" {
		return dto.DraftLayout{}, apperr.Conflict("this draft has no document")
	}
	if outcome(draft) == dto.OutcomeDiscarded {
		return dto.DraftLayout{}, apperr.Conflict("this draft was discarded")
	}
	var edits cvedit.EditSet
	if err := json.Unmarshal(draft.EditSet, &edits); err != nil {
		return dto.DraftLayout{}, fmt.Errorf("decode edit set: %w", err)
	}
	raw, err := s.docs.GetDocument(ctx, userID, draft.DraftDocID, draft.BaseTabID)
	if err != nil {
		return dto.DraftLayout{}, fmt.Errorf("load draft doc: %w", err)
	}
	doc, err := tabcopy.Layout(raw)
	if errors.Is(err, tabcopy.ErrUnsupported) {
		return dto.DraftLayout{}, apperr.Unprocessable(err.Error())
	}
	if err != nil {
		return dto.DraftLayout{}, fmt.Errorf("lay out draft doc: %w", err)
	}
	ds, err := docparse.Parse(raw)
	if err != nil {
		return dto.DraftLayout{}, apperr.Unprocessable("could not read the draft document")
	}
	claim := dto.DraftClaim{ID: draft.ID, UserID: userID, DocID: draft.BaseDocID, TabID: draft.BaseTabID, AchievementIDs: draft.AchievementIDs}
	pl, err := s.planOf(ctx, claim, ds)
	if err != nil {
		return dto.DraftLayout{}, err
	}

	slotAt := baseSlotsByStart(pl, edits)
	out := dto.DraftLayout{Page: doc.Page, Blocks: make([]dto.LayoutBlock, len(doc.Blocks)), Warnings: []string{}}
	for i, b := range doc.Blocks {
		b.SlotID = slotAt[b.StartIndex]
		if sk := ds.Skills; sk != nil && b.StartIndex >= sk.StartIndex && b.StartIndex < sk.EndIndex {
			b.Section = skillsSection
		}
		out.Blocks[i] = b.LayoutBlock
	}
	return out, nil
}

func baseSlotsByStart(pl plan, edits cvedit.EditSet) map[int]string {
	out := map[int]string{}
	if pl.structure.Profile != nil {
		out[pl.structure.Profile.StartIndex] = profileSlotID
	}
	startOf := make(map[string]int, len(pl.structure.Slots))
	for _, slot := range pl.structure.Slots {
		startOf[slot.ID] = slot.StartIndex
	}
	bulletsOf := make(map[string][]cvedit.Bullet, len(edits.Positions))
	for _, pe := range edits.Positions {
		bulletsOf[pe.PositionID] = pe.Bullets
	}
	for _, p := range pl.positions {
		for i, docSlot := range p.slotIDs {
			if bullets := bulletsOf[p.ID]; i < len(bullets) && bullets[i].SlotID != "" {
				out[startOf[docSlot]] = bullets[i].SlotID
			}
		}
	}
	return out
}

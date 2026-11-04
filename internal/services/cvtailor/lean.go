package cvtailor

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
)

func (s *Service) DraftLean(ctx context.Context, userID string, q dto.DraftLeanQuery) ([]dto.BulletLean, error) {
	draft, err := s.store.GetDraft(ctx, userID, q.ID)
	if err != nil {
		return nil, err
	}
	out := []dto.BulletLean{}
	if len(draft.EditSet) == 0 {
		return out, nil
	}
	var edits cvedit.EditSet
	if err := json.Unmarshal(draft.EditSet, &edits); err != nil {
		return nil, fmt.Errorf("decode edit set: %w", err)
	}
	bank, err := s.store.ListPositions(ctx, userID)
	if err != nil {
		return nil, err
	}
	byID, byText := indexLines(bank)
	bankByPosition := map[string][]dto.Achievement{}
	for _, p := range bank {
		bankByPosition[p.ID] = p.Achievements
	}
	baseTexts, err := baseSlotTexts(draft)
	if err != nil {
		return nil, err
	}

	texts := map[string]bool{}
	used := map[string]bool{}
	var positionOf []string
	var replacedFrom []string
	var lineText []string
	for _, p := range edits.Positions {
		for _, a := range bankByPosition[p.PositionID] {
			for _, l := range achievementLines(a) {
				texts[l.Text] = true
			}
		}
		for _, l := range p.Lines() {
			bullet := dto.BulletLean{SlotID: l.SlotID}
			text := ""
			if a, line, ok := linkedLine(l.AchievementIDs, l.VariantID, l.Text, byID, byText); ok {
				bullet.AchievementID = &a.ID
				used[a.ID] = true
				text = line.Text
				texts[text] = true
			}
			old := ""
			if b, ok := baseTexts[l.SlotID]; ok && len(l.AchievementIDs) > 0 && normalizeText(b) != normalizeText(l.Text) {
				old = b
				texts[b] = true
			}
			out = append(out, bullet)
			positionOf = append(positionOf, p.PositionID)
			replacedFrom = append(replacedFrom, old)
			lineText = append(lineText, text)
		}
	}
	if len(texts) == 0 {
		return out, nil
	}
	questions := make([]string, 0, len(texts))
	for _, t := range slices.Sorted(maps.Keys(texts)) {
		questions = append(questions, achievementQuestion(t))
	}
	answers, err := s.asker.Ask(ctx, userID, draft.JobID, questions)
	if err != nil {
		return nil, err
	}
	leanOf := func(text string) float64 {
		ans := answers[achievementQuestion(text)]
		return ans.PYes - ans.PNo
	}

	for i := range out {
		if out[i].AchievementID != nil {
			lean := leanOf(lineText[i])
			out[i].Lean, out[i].State = &lean, suggestionState(lean)
		}
		if replacedFrom[i] != "" {
			lean := leanOf(replacedFrom[i])
			out[i].ReplacedFrom, out[i].ReplacedLean = &replacedFrom[i], &lean
		}
		if out[i].Lean == nil {
			continue
		}
		var best *dto.CloserMatch
		for _, a := range bankByPosition[positionOf[i]] {
			if used[a.ID] {
				continue
			}
			if line, lean, ok := bestLine(a, answers); ok && lean > *out[i].Lean && (best == nil || lean > best.Lean) {
				best = &dto.CloserMatch{AchievementID: a.ID, Text: line.Text, Lean: lean}
			}
		}
		out[i].CloserMatch = best
	}
	return out, nil
}

func baseSlotTexts(draft dto.Draft) (map[string]string, error) {
	out := map[string]string{}
	if len(draft.BaseContent) == 0 || len(draft.EditSet) == 0 {
		return out, nil
	}
	var base baseContent
	if err := json.Unmarshal(draft.BaseContent, &base); err != nil {
		return nil, fmt.Errorf("decode base content: %w", err)
	}
	var edits cvedit.EditSet
	if err := json.Unmarshal(draft.EditSet, &edits); err != nil {
		return nil, fmt.Errorf("decode edit set: %w", err)
	}
	for _, bp := range base.Positions {
		for _, pe := range edits.Positions {
			if pe.PositionID != bp.PositionID {
				continue
			}
			for i, b := range pe.Bullets {
				if i < len(bp.Bullets) {
					out[b.SlotID] = bp.Bullets[i].Text
				}
			}
		}
	}
	return out, nil
}

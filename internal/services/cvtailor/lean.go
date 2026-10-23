package cvtailor

import (
	"context"
	"encoding/json"
	"fmt"

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
	byID := map[string]dto.Achievement{}
	byText := map[string]dto.Achievement{}
	for _, p := range bank {
		for _, a := range p.Achievements {
			byID[a.ID] = a
			byText[normalizeText(a.Text)] = a
		}
	}

	var questions []string
	for _, p := range edits.Positions {
		for _, l := range p.Lines() {
			bullet := dto.BulletLean{SlotID: l.SlotID}
			if a, ok := linkedAchievement(l, byID, byText); ok {
				bullet.AchievementID = &a.ID
				questions = append(questions, achievementQuestion(a.Text))
			}
			out = append(out, bullet)
		}
	}
	if len(questions) == 0 {
		return out, nil
	}
	answers, err := s.asker.Ask(ctx, userID, draft.JobID, questions)
	if err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].AchievementID == nil {
			continue
		}
		ans := answers[achievementQuestion(byID[*out[i].AchievementID].Text)]
		lean := ans.PYes - ans.PNo
		out[i].Lean, out[i].State = &lean, suggestionState(lean)
	}
	return out, nil
}

func linkedAchievement(l cvedit.Line, byID, byText map[string]dto.Achievement) (dto.Achievement, bool) {
	for _, id := range l.AchievementIDs {
		if a, ok := byID[id]; ok {
			return a, true
		}
	}
	a, ok := byText[normalizeText(l.Text)]
	return a, ok
}

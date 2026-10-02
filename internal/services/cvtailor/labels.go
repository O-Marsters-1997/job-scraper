package cvtailor

import (
	"context"
	"slices"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func (s *Service) bulletLabels(ctx context.Context, userID string, in dto.DraftInput, positions []dto.Position) ([]dto.BulletLabel, error) {
	var questions []string
	for _, p := range positions {
		for _, a := range p.Achievements {
			questions = append(questions, achievementQuestion(a.Text))
		}
	}
	answers, err := s.asker.Ask(ctx, userID, in.JobID, questions)
	if err != nil {
		return nil, err
	}
	slots, err := s.slotCounts(ctx, userID, dto.SuggestionsQuery{JobID: in.JobID, DocID: in.DocID, TabID: in.TabID}, positions)
	if err != nil {
		return nil, err
	}
	suggestions := rankSuggestions(positions, answers, slots)
	labels := make([]dto.BulletLabel, len(suggestions))
	for i, sg := range suggestions {
		labels[i] = dto.BulletLabel{
			AchievementID: sg.AchievementID, Preselected: sg.Preselected, Kept: slices.Contains(in.AchievementIDs, sg.AchievementID),
		}
		if ans, ok := answers[achievementQuestion(sg.Text)]; ok {
			labels[i].Answer = &ans
		}
	}
	return labels, nil
}

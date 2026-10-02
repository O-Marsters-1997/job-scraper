package cvtailor

import (
	"context"
	"log/slog"
	"slices"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
)

func (s *Service) bulletLabels(ctx context.Context, userID string, in dto.DraftInput, positions []dto.Position) []dto.BulletLabel {
	var questions []string
	for _, p := range positions {
		for _, a := range p.Achievements {
			questions = append(questions, achievementQuestion(a.Text))
		}
	}
	answers, err := s.asker.Ask(ctx, userID, in.JobID, questions)
	if err != nil {
		slog.WarnContext(ctx, "bullet labels recorded without Jev answers", slog.Any(logger.KeyErr, err))
	}
	slots, err := s.slotCounts(ctx, userID, dto.SuggestionsQuery{JobID: in.JobID, DocID: in.DocID, TabID: in.TabID}, positions)
	if err != nil {
		slog.WarnContext(ctx, "bullet labels recorded with default slot counts", slog.Any(logger.KeyErr, err))
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
	return labels
}

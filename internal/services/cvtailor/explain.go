package cvtailor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
)

const maxExplainJobChars = 6000

// Explain asks Haiku for one line on what the Job wants that a low or
// unclear Achievement does not show. Jev's answer is re-read, never taken
// from the caller.
func (m *Module) Explain(ctx context.Context, userID string, in dto.ExplainInput) (dto.Explanation, error) {
	positions, err := m.store.ListPositions(ctx, userID)
	if err != nil {
		return dto.Explanation{}, err
	}
	bullet, ok := findAchievement(positions, in.AchievementID)
	if !ok {
		return dto.Explanation{}, apperr.NotFound("achievement not found")
	}
	key, err := m.creds.Get(ctx, userID, jev.Provider)
	if errors.Is(err, data.ErrNotFound) {
		return dto.Explanation{}, apperr.Unprocessable("connect an OpenRouter key in Settings, AI to get explanations")
	}
	if err != nil {
		return dto.Explanation{}, fmt.Errorf("load credential: %w", err)
	}
	description, err := m.store.JobDescription(ctx, in.JobID)
	if err != nil {
		return dto.Explanation{}, err
	}
	question := achievementQuestion(bullet.Text)
	answers, err := m.svc.asker.Ask(ctx, userID, in.JobID, []string{question})
	if err != nil {
		return dto.Explanation{}, err
	}
	answer := answers[question]
	if suggestionState(answer.PYes-answer.PNo) == dto.SuggestionFit {
		return dto.Explanation{}, apperr.Conflict("only a low or unclear bullet can be explained")
	}

	res, err := m.editor.Explain(ctx, key, cvedit.ExplainInput{
		Bullet: bullet.Text, JobDescription: truncateRunes(description, maxExplainJobChars),
		PYes: answer.PYes, PNo: answer.PNo, PNotStated: answer.PNotStated,
	})
	slog.InfoContext(ctx, "bullet explanation", slog.String("achievement_id", in.AchievementID), slog.String("job_id", in.JobID),
		slog.Float64("p_yes", answer.PYes), slog.Float64("p_no", answer.PNo), slog.Float64("p_not_stated", answer.PNotStated),
		slog.String("model", cvedit.SuggestModel), slog.String("reply", res.Text), slog.Float64("cost", res.Cost),
		slog.Any(logger.KeyErr, err))
	if err != nil {
		return dto.Explanation{}, suggestError(err)
	}
	return dto.Explanation{Text: res.Text}, nil
}

func findAchievement(positions []dto.Position, id string) (dto.Achievement, bool) {
	for _, p := range positions {
		for _, a := range p.Achievements {
			if a.ID == id {
				return a, true
			}
		}
	}
	return dto.Achievement{}, false
}

func truncateRunes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

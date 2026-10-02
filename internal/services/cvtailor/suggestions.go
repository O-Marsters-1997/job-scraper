package cvtailor

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	defaultSlotCount = 3
	lowFitLean       = -0.2
	matchTopN        = 3
)

type Asker interface {
	Ask(ctx context.Context, userID, jobID string, questions []string) (map[string]dto.Answer, error)
}

func achievementQuestion(text string) string {
	return "Would this job value a candidate who: " + text
}

func bankQuestions(positions []dto.Position) []string {
	var questions []string
	for _, p := range positions {
		for _, a := range p.Achievements {
			questions = append(questions, achievementQuestion(a.Text))
		}
	}
	return questions
}

func (s *Service) Suggestions(ctx context.Context, userID string, q dto.SuggestionsQuery) ([]dto.Suggestion, error) {
	positions, err := s.store.ListPositions(ctx, userID)
	if err != nil {
		return nil, err
	}
	questions := bankQuestions(positions)
	if len(questions) == 0 {
		return []dto.Suggestion{}, nil
	}
	answers, err := s.asker.Ask(ctx, userID, q.JobID, questions)
	if err != nil {
		return nil, err
	}
	slots, err := s.slotCounts(ctx, userID, q, positions)
	if err != nil {
		return nil, err
	}
	return rankSuggestions(positions, answers, slots), nil
}

func (s *Service) slotCounts(ctx context.Context, userID string, q dto.SuggestionsQuery, positions []dto.Position) (map[string]int, error) {
	if q.DocID == "" || q.TabID == "" {
		return nil, nil
	}
	headings, err := s.headings(ctx, userID, dto.CVTabQuery{DocID: q.DocID, TabID: q.TabID}, positions)
	if err != nil {
		return nil, fmt.Errorf("tailoring: slot counts: %w", err)
	}
	counts := map[string]int{}
	for _, h := range headings {
		if h.Confirmed && h.PositionID != nil {
			counts[*h.PositionID] += h.SlotCount
		}
	}
	return counts, nil
}

func rankSuggestions(positions []dto.Position, answers map[string]dto.Answer, slots map[string]int) []dto.Suggestion {
	out := []dto.Suggestion{}
	for _, p := range positions {
		ranked := make([]dto.Suggestion, len(p.Achievements))
		for i, a := range p.Achievements {
			ans := answers[achievementQuestion(a.Text)]
			lean := ans.PYes - ans.PNo
			ranked[i] = dto.Suggestion{
				AchievementID: a.ID, PositionID: p.ID, Text: a.Text, Score: lean, State: suggestionState(lean),
			}
		}
		slices.SortStableFunc(ranked, func(a, b dto.Suggestion) int { return cmp.Compare(b.Score, a.Score) })
		n := defaultSlotCount
		if c := slots[p.ID]; c > 0 {
			n = c
		}
		for i := range ranked {
			ranked[i].Preselected = i < n
		}
		out = append(out, ranked...)
	}
	return out
}

func suggestionState(lean float64) dto.SuggestionState {
	switch {
	case lean > 0:
		return dto.SuggestionFit
	case lean <= lowFitLean:
		return dto.SuggestionLow
	default:
		return dto.SuggestionUnclear
	}
}

func (s *Service) ExperienceMatch(ctx context.Context, userID string, q dto.ExperienceMatchQuery) (dto.ExperienceMatch, error) {
	positions, err := s.store.ListPositions(ctx, userID)
	if err != nil {
		return dto.ExperienceMatch{}, err
	}
	questions := bankQuestions(positions)
	if len(questions) == 0 {
		return dto.ExperienceMatch{}, nil
	}
	answers, err := s.asker.Ask(ctx, userID, q.JobID, questions)
	if err != nil {
		return dto.ExperienceMatch{}, err
	}
	score := topLeanMean(questions, answers)
	return dto.ExperienceMatch{Score: &score}, nil
}

func topLeanMean(questions []string, answers map[string]dto.Answer) float64 {
	leans := make([]float64, len(questions))
	for i, q := range questions {
		leans[i] = answers[q].PYes - answers[q].PNo
	}
	slices.SortFunc(leans, func(a, b float64) int { return cmp.Compare(b, a) })
	top := leans[:min(matchTopN, len(leans))]
	var sum float64
	for _, l := range top {
		sum += l
	}
	return sum / float64(len(top))
}

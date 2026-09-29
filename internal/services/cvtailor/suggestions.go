package cvtailor

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const defaultSlotCount = 3

// Asker answers arbitrary questions about a Job, keyed by question text.
type Asker interface {
	Ask(ctx context.Context, userID, jobID string, questions []string) (map[string]dto.Answer, error)
}

func achievementQuestion(text string) string {
	return "Would this job value a candidate who: " + text
}

// Suggestions ranks the User's Achievements for the Job by P(yes) x confidence
// and preselects the top N per Position: its base CV slot count, else 3.
// docID and tabID may be empty.
func (s *Service) Suggestions(ctx context.Context, userID string, q dto.SuggestionsQuery) ([]dto.Suggestion, error) {
	positions, err := s.store.ListPositions(ctx, userID)
	if err != nil {
		return nil, err
	}
	var questions []string
	for _, p := range positions {
		for _, a := range p.Achievements {
			questions = append(questions, achievementQuestion(a.Text))
		}
	}
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
			ranked[i] = dto.Suggestion{
				AchievementID: a.ID, PositionID: p.ID, Text: a.Text, Score: ans.PYes * ans.Confidence,
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

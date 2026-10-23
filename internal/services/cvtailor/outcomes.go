package cvtailor

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
)

func (s *Service) keepOutcomes(ctx context.Context, claim dto.DraftClaim) ([]dto.AchievementOutcome, error) {
	draft, err := s.store.GetDraft(ctx, claim.UserID, claim.ID)
	if err != nil {
		return nil, err
	}
	bank, err := s.store.ListPositions(ctx, claim.UserID)
	if err != nil {
		return nil, err
	}
	var edits cvedit.EditSet
	if len(draft.EditSet) > 0 {
		if err := json.Unmarshal(draft.EditSet, &edits); err != nil {
			return nil, fmt.Errorf("decode edit set: %w", err)
		}
	}
	source := map[string]bool{}
	if len(draft.BaseContent) > 0 {
		var stored baseContent
		if err := json.Unmarshal(draft.BaseContent, &stored); err != nil {
			return nil, fmt.Errorf("decode base content: %w", err)
		}
		for _, p := range stored.Positions {
			for _, b := range p.Bullets {
				source[normalizeText(b.Text)] = true
			}
		}
	}
	answers, err := s.asker.Ask(ctx, claim.UserID, claim.JobID, bankQuestions(bank))
	if err != nil {
		answers = nil
	}
	return achievementOutcomes(bank, answers, edits, source), nil
}

func achievementOutcomes(bank []dto.Position, answers map[string]dto.Answer, edits cvedit.EditSet, source map[string]bool) []dto.AchievementOutcome {
	linesOf := make(map[string][]cvedit.Line, len(edits.Positions))
	for _, pe := range edits.Positions {
		linesOf[pe.PositionID] = pe.Lines()
	}
	var out []dto.AchievementOutcome
	for _, p := range bank {
		idByText := make(map[string]string, len(p.Achievements))
		leans := map[string]float64{}
		for _, a := range p.Achievements {
			idByText[normalizeText(a.Text)] = a.ID
			if ans, ok := answers[achievementQuestion(a.Text)]; ok {
				leans[a.ID] = ans.PYes - ans.PNo
			}
		}
		inFinal := map[string]bool{}
		verbatim := map[string]bool{}
		var keptLeans []float64
		for _, l := range linesOf[p.ID] {
			ids := slices.Clone(l.AchievementIDs)
			if id, ok := idByText[normalizeText(l.Text)]; ok {
				ids = append(ids, id)
				verbatim[id] = true
			}
			best, answered := 0.0, false
			for _, id := range ids {
				inFinal[id] = true
				if lean, ok := leans[id]; ok && (!answered || lean > best) {
					best, answered = lean, true
				}
			}
			if answered {
				keptLeans = append(keptLeans, best)
			}
		}
		for _, a := range p.Achievements {
			lean, answered := leans[a.ID]
			o := dto.AchievementOutcome{
				AchievementID: a.ID,
				OnSourceCV:    source[normalizeText(a.Text)],
				InFinalCV:     inFinal[a.ID],
				Reworded:      inFinal[a.ID] && !verbatim[a.ID],
			}
			if answered {
				ans := answers[achievementQuestion(a.Text)]
				o.PYes, o.PNo = &ans.PYes, &ans.PNo
				o.OutscoredKeptButLeftOut = !o.InFinalCV && slices.ContainsFunc(keptLeans, func(kept float64) bool { return lean > kept })
			}
			out = append(out, o)
		}
	}
	return out
}

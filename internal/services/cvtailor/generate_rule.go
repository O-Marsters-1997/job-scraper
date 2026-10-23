package cvtailor

import (
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
)

type Replacement struct {
	PositionID    string
	SlotIndex     int
	AchievementID string
}

type ReplacementRule interface {
	Replacements(positions []dto.Position, content dto.DraftContent, answers map[string]dto.Answer) []Replacement
}

func buildsFromSource(mode dto.DraftMode, achievementIDs []string) bool {
	return mode == dto.DraftModeAsIs || len(achievementIDs) == 0
}

type leanRule struct{}

var _ ReplacementRule = leanRule{}

func (leanRule) Replacements(positions []dto.Position, content dto.DraftContent, answers map[string]dto.Answer) []Replacement {
	byPosition := make(map[string]dto.Position, len(positions))
	byText := map[string]dto.Achievement{}
	byID := map[string]dto.Achievement{}
	for _, p := range positions {
		byPosition[p.ID] = p
		for _, a := range p.Achievements {
			byText[normalizeText(a.Text)] = a
			byID[a.ID] = a
		}
	}
	leanOf := func(a dto.Achievement) (float64, bool) {
		ans, ok := answers[achievementQuestion(a.Text)]
		return ans.PYes - ans.PNo, ok
	}
	linked := func(b dto.DraftBullet) (dto.Achievement, bool) {
		for _, id := range b.AchievementIDs {
			if a, ok := byID[id]; ok {
				return a, true
			}
		}
		a, ok := byText[normalizeText(b.Text)]
		return a, ok
	}

	used := map[string]bool{}
	for _, cp := range content.Positions {
		for _, b := range cp.Bullets {
			if a, ok := linked(b); ok {
				used[a.ID] = true
			}
		}
	}

	var out []Replacement
	for _, cp := range content.Positions {
		bank, ok := byPosition[cp.PositionID]
		if !ok {
			continue
		}
		for i, b := range cp.Bullets {
			a, ok := linked(b)
			if !ok {
				continue
			}
			lean, ok := leanOf(a)
			if !ok || lean > lowFitLean {
				continue
			}
			best, bestLean := "", lean
			for _, c := range bank.Achievements {
				if used[c.ID] {
					continue
				}
				if l, ok := leanOf(c); ok && l > bestLean {
					best, bestLean = c.ID, l
				}
			}
			if best == "" {
				continue
			}
			used[best] = true
			out = append(out, Replacement{PositionID: cp.PositionID, SlotIndex: i, AchievementID: best})
		}
	}
	return out
}

func (pl plan) withReplacements(positions []dto.Position, reps []Replacement) cvedit.EditSet {
	edits := pl.keepAll()
	texts := map[string]string{}
	for _, p := range positions {
		for _, a := range p.Achievements {
			texts[a.ID] = a.Text
		}
	}
	index := make(map[string]int, len(pl.positions))
	for i, p := range pl.positions {
		index[p.ID] = i
	}
	for _, r := range reps {
		i := index[r.PositionID]
		text := texts[r.AchievementID]
		pl.positions[i].texts[r.AchievementID] = text
		edits.Positions[i].Bullets[r.SlotIndex] = cvedit.Bullet{
			SlotID: pl.positions[i].slotIDs[r.SlotIndex], Text: text, AchievementIDs: []string{r.AchievementID},
		}
	}
	return edits
}

package cvtailor

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/linefit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/tabcopy"
)

const AppendSlot = -1

const maxFreeLines = 1

type Replacement struct {
	PositionID     string
	SlotIndex      int
	AchievementIDs []string
}

type RuleInput struct {
	Positions []dto.Position
	Content   dto.DraftContent
	Answers   map[string]dto.Answer
	Lines     func(positionID, text string) int
}

type ReplacementRule interface {
	Replacements(in RuleInput) []Replacement
}

func buildsFromSource(mode dto.DraftMode, achievementIDs []string) bool {
	return mode == dto.DraftModeAsIs || len(achievementIDs) == 0
}

type leanRule struct{}

var _ ReplacementRule = leanRule{}

func (leanRule) Replacements(in RuleInput) []Replacement {
	byPosition := make(map[string]dto.Position, len(in.Positions))
	byText := map[string]dto.Achievement{}
	byID := map[string]dto.Achievement{}
	for _, p := range in.Positions {
		byPosition[p.ID] = p
		for _, a := range p.Achievements {
			byText[normalizeText(a.Text)] = a
			byID[a.ID] = a
		}
	}
	leanOf := func(a dto.Achievement) (float64, bool) {
		ans, ok := in.Answers[achievementQuestion(a.Text)]
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
	for _, cp := range in.Content.Positions {
		for _, b := range cp.Bullets {
			if a, ok := linked(b); ok {
				used[a.ID] = true
			}
		}
	}

	var out []Replacement
	for _, cp := range in.Content.Positions {
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
			var pool []dto.Achievement
			for _, c := range bank.Achievements {
				if l, ok := leanOf(c); ok && !used[c.ID] && l > lean {
					pool = append(pool, c)
				}
			}
			slices.SortStableFunc(pool, func(x, y dto.Achievement) int {
				lx, _ := leanOf(x)
				ly, _ := leanOf(y)
				return cmp.Compare(ly, lx)
			})
			free := in.Lines(cp.PositionID, b.Text)
			var ids []string
			for _, c := range pool {
				if n := in.Lines(cp.PositionID, c.Text); n <= free {
					ids = append(ids, c.ID)
					free -= n
				}
			}
			if len(ids) == 0 || free > maxFreeLines {
				continue
			}
			for _, id := range ids {
				used[id] = true
			}
			out = append(out, Replacement{PositionID: cp.PositionID, SlotIndex: i, AchievementIDs: ids})
		}
	}
	return out
}

func sourceLines(raw json.RawMessage, pl plan) (func(positionID, text string) int, error) {
	doc, err := tabcopy.Layout(raw)
	if errors.Is(err, tabcopy.ErrUnsupported) {
		return func(string, string) int { return 1 }, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lay out CV tab: %w", err)
	}
	blockAt := make(map[int]tabcopy.Block, len(doc.Blocks))
	for _, b := range doc.Blocks {
		blockAt[b.StartIndex] = b
	}
	startOf := make(map[string]int, len(pl.structure.Slots))
	for _, slot := range pl.structure.Slots {
		startOf[slot.ID] = slot.StartIndex
	}
	type measure struct {
		style linefit.Style
		width float64
	}
	byPosition := map[string]measure{}
	for _, p := range pl.positions {
		if len(p.slotIDs) == 0 {
			continue
		}
		b, ok := blockAt[startOf[p.slotIDs[0]]]
		if !ok || len(b.Runs) == 0 {
			continue
		}
		r := b.Runs[0]
		byPosition[p.ID] = measure{
			style: linefit.Style{Font: r.Font, Size: r.Size, Bold: r.Bold, Italic: r.Italic},
			width: linefit.Width(doc.Page, b.LayoutBlock),
		}
	}
	return func(positionID, text string) int {
		m, ok := byPosition[positionID]
		if !ok {
			return 1
		}
		return linefit.Lines(text, m.style, m.width)
	}, nil
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
		i, ok := index[r.PositionID]
		if !ok || len(r.AchievementIDs) == 0 {
			continue
		}
		pe := &edits.Positions[i]
		ids := r.AchievementIDs
		after := ""
		if r.SlotIndex == AppendSlot {
			if n := len(pe.Bullets); n > 0 {
				after = pe.Bullets[n-1].SlotID
			}
			follows := func(a cvedit.AddedBullet) bool { return a.After == after }
			for at := slices.IndexFunc(pe.Added, follows); at >= 0; at = slices.IndexFunc(pe.Added, follows) {
				after = pe.Added[at].SlotID
			}
		} else {
			text := texts[ids[0]]
			pl.positions[i].texts[ids[0]] = text
			slotID := pl.positions[i].slotIDs[r.SlotIndex]
			pe.Bullets[r.SlotIndex] = cvedit.Bullet{SlotID: slotID, Text: text, AchievementIDs: []string{ids[0]}}
			after, ids = slotID, ids[1:]
		}
		for _, id := range ids {
			text := texts[id]
			pl.positions[i].texts[id] = text
			added := cvedit.AddedBullet{
				SlotID: fmt.Sprintf("%s%s-%d", addedSlotPrefix, r.PositionID, len(pe.Added)),
				After:  after, Text: text, AchievementIDs: []string{id},
			}
			pe.Added = append(pe.Added, added)
			after = added.SlotID
		}
	}
	return edits
}

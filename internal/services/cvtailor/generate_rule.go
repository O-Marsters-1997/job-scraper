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

// Pick's empty VariantID means the Achievement's default text.
type Pick struct {
	AchievementID string
	VariantID     string
}

type Replacement struct {
	PositionID string
	SlotIndex  int
	Picks      []Pick
}

type bankLine struct {
	Pick
	Text string
}

func achievementLines(a dto.Achievement) []bankLine {
	out := make([]bankLine, 0, 1+len(a.Variants))
	out = append(out, bankLine{Pick{AchievementID: a.ID}, a.Text})
	for _, v := range a.Variants {
		out = append(out, bankLine{Pick{AchievementID: a.ID, VariantID: v.ID}, v.Text})
	}
	return out
}

func lineQuestions(positions []dto.Position) []string {
	var questions []string
	for _, p := range positions {
		for _, a := range p.Achievements {
			for _, l := range achievementLines(a) {
				questions = append(questions, achievementQuestion(l.Text))
			}
		}
	}
	return questions
}

func indexLines(positions []dto.Position) (byID map[string]dto.Achievement, byText map[string]bankLine) {
	byID, byText = map[string]dto.Achievement{}, map[string]bankLine{}
	for _, p := range positions {
		for _, a := range p.Achievements {
			byID[a.ID] = a
			for _, l := range achievementLines(a) {
				byText[normalizeText(l.Text)] = l
			}
		}
	}
	return byID, byText
}

func linkedLine(ids []string, variantID, text string, byID map[string]dto.Achievement, byText map[string]bankLine) (dto.Achievement, bankLine, bool) {
	for _, id := range ids {
		a, ok := byID[id]
		if !ok {
			continue
		}
		lines := achievementLines(a)
		at := slices.IndexFunc(lines, func(l bankLine) bool {
			if variantID != "" {
				return l.VariantID == variantID
			}
			return normalizeText(l.Text) == normalizeText(text)
		})
		return a, lines[max(at, 0)], true
	}
	l, ok := byText[normalizeText(text)]
	return byID[l.AchievementID], l, ok
}

func bestLine(a dto.Achievement, answers map[string]dto.Answer) (line bankLine, lean float64, ok bool) {
	for _, l := range achievementLines(a) {
		ans, answered := answers[achievementQuestion(l.Text)]
		if v := ans.PYes - ans.PNo; answered && (!ok || v > lean) {
			line, lean, ok = l, v, true
		}
	}
	return line, lean, ok
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
	byID, byText := indexLines(in.Positions)
	for _, p := range in.Positions {
		byPosition[p.ID] = p
	}
	leanOf := func(l bankLine) (float64, bool) {
		ans, ok := in.Answers[achievementQuestion(l.Text)]
		return ans.PYes - ans.PNo, ok
	}
	linked := func(b dto.DraftBullet) (dto.Achievement, bankLine, bool) {
		return linkedLine(b.AchievementIDs, "", b.Text, byID, byText)
	}

	used := map[string]bool{}
	for _, cp := range in.Content.Positions {
		for _, b := range cp.Bullets {
			if a, _, ok := linked(b); ok {
				used[a.ID] = true
			}
		}
	}

	type scored struct {
		bankLine
		lean float64
	}
	var out []Replacement
	for _, cp := range in.Content.Positions {
		bank, ok := byPosition[cp.PositionID]
		if !ok {
			continue
		}
		for i, b := range cp.Bullets {
			a, current, ok := linked(b)
			if !ok {
				continue
			}
			lean, ok := leanOf(current)
			if !ok {
				continue
			}
			var pool []scored
			consider := func(c dto.Achievement) {
				if l, v, ok := bestLine(c, in.Answers); ok && v > lean {
					pool = append(pool, scored{l, v})
				}
			}
			consider(a)
			if lean <= lowFitLean {
				for _, c := range bank.Achievements {
					if !used[c.ID] {
						consider(c)
					}
				}
			}
			slices.SortStableFunc(pool, func(x, y scored) int { return cmp.Compare(y.lean, x.lean) })
			free := in.Lines(cp.PositionID, b.Text)
			var picks []Pick
			for _, c := range pool {
				if n := in.Lines(cp.PositionID, c.Text); n <= free {
					picks = append(picks, c.Pick)
					free -= n
				}
			}
			if len(picks) == 0 || free > maxFreeLines {
				continue
			}
			for _, p := range picks {
				used[p.AchievementID] = true
			}
			out = append(out, Replacement{PositionID: cp.PositionID, SlotIndex: i, Picks: picks})
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
	texts := map[Pick]string{}
	for _, p := range positions {
		for _, a := range p.Achievements {
			for _, l := range achievementLines(a) {
				texts[l.Pick] = l.Text
			}
		}
	}
	index := make(map[string]int, len(pl.positions))
	for i, p := range pl.positions {
		index[p.ID] = i
	}
	for _, r := range reps {
		i, ok := index[r.PositionID]
		if !ok || len(r.Picks) == 0 {
			continue
		}
		pe := &edits.Positions[i]
		picks := r.Picks
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
			pick := picks[0]
			text := texts[pick]
			pl.positions[i].texts[pick.AchievementID] = text
			slotID := pl.positions[i].slotIDs[r.SlotIndex]
			pe.Bullets[r.SlotIndex] = cvedit.Bullet{SlotID: slotID, Text: text, AchievementIDs: []string{pick.AchievementID}, VariantID: pick.VariantID}
			after, picks = slotID, picks[1:]
		}
		for _, pick := range picks {
			text := texts[pick]
			pl.positions[i].texts[pick.AchievementID] = text
			added := cvedit.AddedBullet{
				SlotID: fmt.Sprintf("%s%s-%d", addedSlotPrefix, r.PositionID, len(pe.Added)),
				After:  after, Text: text, AchievementIDs: []string{pick.AchievementID}, VariantID: pick.VariantID,
			}
			pe.Added = append(pe.Added, added)
			after = added.SlotID
		}
	}
	return edits
}

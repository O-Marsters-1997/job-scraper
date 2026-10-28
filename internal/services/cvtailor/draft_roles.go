package cvtailor

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/docparse"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/linefit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/tabcopy"
)

const pageCountCheck = "page_count"

type pageGeometry struct {
	page     dto.LayoutPage
	fallback tabcopy.Block
}

func (g pageGeometry) lines(text string, b *tabcopy.Block) int {
	if b == nil {
		b = &g.fallback
	}
	style := linefit.Style{Font: "Arial", Size: 11}
	longest := 0
	for _, r := range b.Runs {
		if len(r.Text) > longest {
			longest = len(r.Text)
			style = linefit.Style{Font: r.Font, Size: r.Size, Bold: r.Bold, Italic: r.Italic}
		}
	}
	return linefit.Lines(text, style, linefit.Width(g.page, b.LayoutBlock))
}

func (s *Service) DraftRoles(ctx context.Context, userID string, q dto.DraftQuery) (dto.DraftRoles, error) {
	draft, err := s.store.GetDraft(ctx, userID, q.ID)
	if err != nil {
		return dto.DraftRoles{}, err
	}
	if len(draft.EditSet) == 0 {
		return dto.DraftRoles{}, apperr.Conflict("this draft has no edits yet")
	}
	if outcome(draft) == dto.OutcomeDiscarded {
		return dto.DraftRoles{}, apperr.Conflict("this draft was discarded")
	}
	var edits cvedit.EditSet
	if err := json.Unmarshal(draft.EditSet, &edits); err != nil {
		return dto.DraftRoles{}, fmt.Errorf("decode edit set: %w", err)
	}
	raw, err := s.docs.GetDocument(ctx, userID, draft.BaseDocID, draft.BaseTabID)
	if err != nil {
		return dto.DraftRoles{}, fmt.Errorf("load CV tab: %w", err)
	}
	doc, err := tabcopy.Layout(raw)
	if errors.Is(err, tabcopy.ErrUnsupported) {
		return dto.DraftRoles{}, apperr.Unprocessable(err.Error())
	}
	if err != nil {
		return dto.DraftRoles{}, fmt.Errorf("lay out CV tab: %w", err)
	}
	ds, err := docparse.Parse(raw)
	if err != nil {
		return dto.DraftRoles{}, fmt.Errorf("parse CV tab: %w", err)
	}
	bank, err := s.store.ListPositions(ctx, userID)
	if err != nil {
		return dto.DraftRoles{}, err
	}
	mappings, err := s.linkedMappings(ctx, userID, draft.BaseDocID, draft.BaseTabID, ds, bank)
	if err != nil {
		return dto.DraftRoles{}, err
	}
	leans, err := s.DraftLean(ctx, userID, dto.DraftLeanQuery{ID: draft.ID})
	if err != nil {
		return dto.DraftRoles{}, err
	}

	geo := pageGeometry{page: doc.Page}
	blockAt := make(map[int]*tabcopy.Block, len(doc.Blocks))
	for i := range doc.Blocks {
		blockAt[doc.Blocks[i].StartIndex] = &doc.Blocks[i]
	}
	if len(ds.Slots) > 0 && blockAt[ds.Slots[0].StartIndex] != nil {
		geo.fallback = *blockAt[ds.Slots[0].StartIndex]
	}
	bankOf := make(map[string]dto.Position, len(bank))
	for _, p := range bank {
		bankOf[p.ID] = p
	}
	leanOf := make(map[string]dto.BulletLean, len(leans))
	used := map[string]bool{}
	for _, bl := range leans {
		leanOf[bl.SlotID] = bl
		if bl.AchievementID != nil {
			used[*bl.AchievementID] = true
		}
	}

	slotsOf := slotsByPosition(ds, mappings)
	out := dto.DraftRoles{
		DraftID: draft.ID,
		Kept:    outcome(draft) == dto.OutcomeKept,
		Spills: slices.ContainsFunc(draft.Findings, func(f dto.DraftFinding) bool {
			return f.Check == pageCountCheck && f.Severity == string(checks.Block)
		}),
		Roles: []dto.DraftRole{},
	}
	sourceTexts := map[string]map[string]bool{}
	for _, pe := range edits.Positions {
		slots := slotsOf[pe.PositionID]
		pos := bankOf[pe.PositionID]
		role := dto.DraftRole{PositionID: &pe.PositionID, Employer: pos.Employer, Title: pos.Title, Bullets: []dto.RoleBullet{}, Bank: []dto.RoleBullet{}, Excluded: []dto.RoleBullet{}}
		onSource := map[string]bool{}
		for _, sl := range slots {
			onSource[normalizeText(sl.Text)] = true
			role.Budget += geo.lines(sl.Text, blockAt[sl.StartIndex])
		}
		sourceTexts[pe.PositionID] = onSource
		for i, l := range pe.Lines() {
			var b *tabcopy.Block
			if len(slots) > 0 {
				b = blockAt[slots[min(i, len(slots)-1)].StartIndex]
			}
			bl := leanOf[l.SlotID]
			role.Bullets = append(role.Bullets, dto.RoleBullet{
				SlotID: l.SlotID, AchievementID: bl.AchievementID, Text: l.Text, Lines: geo.lines(l.Text, b),
				Lean: bl.Lean, State: bl.State, OnSource: onSource[normalizeText(l.Text)], ReplacedFrom: bl.ReplacedFrom,
			})
		}
		role.Excluded = excludedBase(role.Bullets, slots, pos, geo, blockAt)
		out.Roles = append(out.Roles, role)
	}
	blockOf := func(positionID string) *tabcopy.Block {
		if slots := slotsOf[positionID]; len(slots) > 0 {
			return blockAt[slots[0].StartIndex]
		}
		return nil
	}
	if err := s.fillBank(ctx, userID, draft.JobID, out.Roles, bankOf, used, sourceTexts, blockOf, geo); err != nil {
		return dto.DraftRoles{}, err
	}
	for i := range out.Roles {
		foldExcludedBank(&out.Roles[i])
	}
	linked := map[string]bool{}
	for _, m := range mappings {
		linked[m.HeadingText] = m.PositionID != nil
	}
	roleAt := map[int]int{}
	for _, sl := range ds.Slots {
		if sl.HeadingIndex < 0 || linked[ds.Headings[sl.HeadingIndex].Text] {
			continue
		}
		i, ok := roleAt[sl.HeadingIndex]
		if !ok {
			i = len(out.Roles)
			roleAt[sl.HeadingIndex] = i
			out.Roles = append(out.Roles, dto.DraftRole{Title: ds.Headings[sl.HeadingIndex].Text, Bullets: []dto.RoleBullet{}, Bank: []dto.RoleBullet{}, Excluded: []dto.RoleBullet{}})
		}
		n := geo.lines(sl.Text, blockAt[sl.StartIndex])
		out.Roles[i].Budget += n
		out.Roles[i].Bullets = append(out.Roles[i].Bullets, dto.RoleBullet{Text: sl.Text, Lines: n, OnSource: true})
	}
	return out, nil
}

func excludedBase(bullets []dto.RoleBullet, slots []docparse.Slot, pos dto.Position, geo pageGeometry, blockAt map[int]*tabcopy.Block) []dto.RoleBullet {
	present := map[string]bool{}
	for _, b := range bullets {
		present[normalizeText(b.Text)] = true
		if b.ReplacedFrom != nil {
			present[normalizeText(*b.ReplacedFrom)] = true
		}
	}
	out := []dto.RoleBullet{}
	for _, sl := range slots {
		if present[normalizeText(sl.Text)] {
			continue
		}
		b := dto.RoleBullet{Text: sl.Text, Lines: geo.lines(sl.Text, blockAt[sl.StartIndex]), OnSource: true}
		for _, a := range pos.Achievements {
			if normalizeText(a.Text) == normalizeText(sl.Text) {
				b.AchievementID = &a.ID
				break
			}
		}
		out = append(out, b)
	}
	return out
}

func foldExcludedBank(r *dto.DraftRole) {
	for i, ex := range r.Excluded {
		if ex.AchievementID == nil {
			continue
		}
		at := slices.IndexFunc(r.Bank, func(b dto.RoleBullet) bool {
			return b.AchievementID != nil && *b.AchievementID == *ex.AchievementID
		})
		if at < 0 {
			continue
		}
		r.Excluded[i].Lean, r.Excluded[i].State = r.Bank[at].Lean, r.Bank[at].State
		r.Bank = slices.Delete(r.Bank, at, at+1)
	}
}

func (s *Service) fillBank(ctx context.Context, userID, jobID string, roles []dto.DraftRole, bankOf map[string]dto.Position, used map[string]bool, sourceTexts map[string]map[string]bool, blockOf func(positionID string) *tabcopy.Block, geo pageGeometry) error {
	texts := map[string]bool{}
	for _, r := range roles {
		for _, a := range bankOf[*r.PositionID].Achievements {
			if !used[a.ID] {
				texts[a.Text] = true
			}
		}
	}
	if len(texts) == 0 {
		return nil
	}
	var questions []string
	for _, t := range slices.Sorted(maps.Keys(texts)) {
		questions = append(questions, achievementQuestion(t))
	}
	answers, err := s.asker.Ask(ctx, userID, jobID, questions)
	if err != nil {
		return err
	}
	for i := range roles {
		r := &roles[i]
		b := blockOf(*r.PositionID)
		for _, a := range bankOf[*r.PositionID].Achievements {
			if used[a.ID] {
				continue
			}
			ans := answers[achievementQuestion(a.Text)]
			lean := ans.PYes - ans.PNo
			id := a.ID
			r.Bank = append(r.Bank, dto.RoleBullet{
				AchievementID: &id, Text: a.Text, Lines: geo.lines(a.Text, b), Lean: &lean, State: suggestionState(lean),
				OnSource: sourceTexts[*r.PositionID][normalizeText(a.Text)],
			})
		}
		slices.SortStableFunc(r.Bank, func(x, y dto.RoleBullet) int { return cmp.Compare(*y.Lean, *x.Lean) })
	}
	return nil
}

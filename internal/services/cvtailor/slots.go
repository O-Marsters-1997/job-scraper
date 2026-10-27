package cvtailor

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/docparse"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/docedit"
)

const (
	profileSlotID = "profile"

	addedSlotPrefix   = "new-"
	maxAddedSlotIDLen = 64
)

// SaveDraftSlots writes the User's bullet and profile edits into the Draft's Doc and
// re-runs the checks. Findings never block a save.
func (s *Service) SaveDraftSlots(ctx context.Context, userID string, in dto.DraftSlotsInput) (dto.Draft, error) {
	draft, err := s.store.GetDraft(ctx, userID, in.ID)
	if err != nil {
		return dto.Draft{}, err
	}
	if draft.Status != statusReady || draft.DraftDocID == "" {
		return dto.Draft{}, apperr.Conflict("only a ready draft can be edited")
	}
	if outcome(draft) == dto.OutcomeDiscarded {
		return dto.Draft{}, apperr.Conflict("this draft was discarded")
	}
	var edits cvedit.EditSet
	if err := json.Unmarshal(draft.EditSet, &edits); err != nil {
		return dto.Draft{}, fmt.Errorf("decode edit set: %w", err)
	}

	bank, err := s.store.ListPositions(ctx, userID)
	if err != nil {
		return dto.Draft{}, err
	}
	roles := edits.Roles
	if in.Roles != nil {
		roles = in.Roles
	}
	cited, err := citedByReplacements(bank, edits, in.Positions, roles)
	if err != nil {
		return dto.Draft{}, err
	}
	chosen := slices.Clone(draft.AchievementIDs)
	for _, id := range cited {
		if !slices.Contains(chosen, id) {
			chosen = append(chosen, id)
		}
	}

	claim := dto.DraftClaim{ID: draft.ID, UserID: userID, JobID: draft.JobID, DocID: draft.BaseDocID, TabID: draft.BaseTabID, AchievementIDs: chosen, Roles: roles, Mode: draft.Mode}
	basePlan, err := s.plan(ctx, claim, claim.DocID)
	if err != nil {
		return dto.Draft{}, err
	}
	docClaim, err := s.withinRoles(ctx, claim, edits.Roles)
	if err != nil {
		return dto.Draft{}, err
	}
	docPlan, err := s.plan(ctx, docClaim, draft.DraftDocID)
	if err != nil {
		return dto.Draft{}, err
	}
	before := edits.Profile
	var layout roleLayout
	rebuilt := false
	if in.Roles != nil {
		edits, layout, rebuilt, err = s.applyRoles(ctx, claim, basePlan.structure, edits, in.Roles, bank)
		if err != nil {
			return dto.Draft{}, err
		}
	}
	edits, err = basePlan.replaceBullets(edits, in.Positions)
	if err != nil {
		return dto.Draft{}, err
	}
	edits, err = basePlan.overlay(edits, in.Slots)
	if err != nil {
		return dto.Draft{}, err
	}
	edits, err = basePlan.overlaySkills(edits, in.Skills)
	if err != nil {
		return dto.Draft{}, err
	}
	docEdits := cvedit.EditSet{Positions: edits.Positions}
	if edits.Profile != before {
		docEdits.Profile = edits.Profile
	}
	if len(in.Skills) > 0 && !skillsEqual(edits.Skills, cvedit.SkillGroups(docPlan.structure.Skills)) {
		docEdits.Skills = edits.Skills
	}
	headings := basePlan.structure.Headings
	switch {
	case rebuilt:
		var ds docparse.DocStructure
		if ds, err = s.rebuildDoc(ctx, claim, draft.DraftDocID, layout, edits); err == nil {
			headings = ds.Headings
		}
	default:
		if edits.Roles != nil {
			headings = docPlan.structure.Headings
		}
		err = s.applyEdits(ctx, userID, draft.DraftDocID, docPlan, docEdits)
	}
	if err != nil {
		return dto.Draft{}, err
	}

	basePages, draftPages, draftPDF, err := s.pageCounts(ctx, claim, draft.DraftDocID)
	if err != nil {
		return dto.Draft{}, err
	}
	editSet, err := json.Marshal(edits)
	if err != nil {
		return dto.Draft{}, fmt.Errorf("marshal edit set: %w", err)
	}
	checked := basePlan.draft(edits, basePages, draftPages)
	if len(in.Skills) > 0 {
		checked.BaseSkills = cvedit.CheckLines(edits.Skills)
	}
	checked.Parse = parseInput(ctx, draftPDF, headings)
	findings := toDraftFindings(checks.Run(checked))
	if err := s.store.SetDraftEdits(ctx, userID, draft.ID, editSet, findings, citedAchievements(bank, edits, draft.AchievementIDs)); err != nil {
		return dto.Draft{}, err
	}
	if outcome(draft) == dto.OutcomeKept {
		outcomes, err := s.keepOutcomes(ctx, claim)
		if err != nil {
			return dto.Draft{}, fmt.Errorf("compute outcomes: %w", err)
		}
		if err := s.store.ReplaceAchievementOutcomes(ctx, claim, outcomes); err != nil {
			return dto.Draft{}, err
		}
	}
	return s.GetDraft(ctx, userID, dto.DraftQuery{ID: draft.ID})
}

func citedByReplacements(bank []dto.Position, edits cvedit.EditSet, reps []dto.DraftPosition, roles []string) ([]string, error) {
	owner := map[string]string{}
	for _, p := range bank {
		for _, a := range p.Achievements {
			owner[a.ID] = p.ID
		}
	}
	var cited []string
	for _, r := range reps {
		if !slices.ContainsFunc(edits.Positions, func(pe cvedit.PositionEdit) bool { return pe.PositionID == r.PositionID }) && !slices.Contains(roles, r.PositionID) {
			return nil, apperr.Invalid("unknown position")
		}
		if len(r.Bullets) == 0 {
			return nil, apperr.Invalid("a position keeps at least one bullet")
		}
		for _, b := range r.Bullets {
			for _, id := range b.AchievementIDs {
				if owner[id] != r.PositionID {
					return nil, apperr.Invalid("unknown achievement")
				}
				cited = append(cited, id)
			}
		}
	}
	return cited, nil
}

func (pl plan) replaceBullets(edits cvedit.EditSet, reps []dto.DraftPosition) (cvedit.EditSet, error) {
	slotIDs := pl.slotIDs()
	byID := pl.byID()
	for _, r := range reps {
		ids := slotIDs[r.PositionID]
		at := slices.IndexFunc(edits.Positions, func(pe cvedit.PositionEdit) bool { return pe.PositionID == r.PositionID })
		if at < 0 {
			return edits, apperr.Invalid("unknown position")
		}
		if len(r.Bullets) < len(ids) {
			return edits, apperr.Invalid("a base bullet cannot be removed")
		}
		pe := cvedit.PositionEdit{PositionID: r.PositionID}
		after := ""
		for i, b := range r.Bullets {
			text := strings.TrimSpace(b.Text)
			if text == "" || strings.ContainsAny(text, "\r\n") {
				return edits, apperr.Invalid("a bullet must be one non-empty line")
			}
			if i < len(ids) {
				keep := len(b.AchievementIDs) == 0 && text == byID[r.PositionID].SlotTexts[i]
				pe.Bullets = append(pe.Bullets, cvedit.Bullet{Keep: keep, Text: text, AchievementIDs: slices.Clone(b.AchievementIDs), SlotID: ids[i]})
				after = ids[i]
				continue
			}
			added := cvedit.AddedBullet{SlotID: fmt.Sprintf("%s%s-%d", addedSlotPrefix, r.PositionID, i), After: after, Text: text, AchievementIDs: slices.Clone(b.AchievementIDs)}
			pe.Added = append(pe.Added, added)
			after = added.SlotID
		}
		edits.Positions = slices.Clone(edits.Positions)
		edits.Positions[at] = pe
	}
	return edits, nil
}

func citedAchievements(bank []dto.Position, edits cvedit.EditSet, previous []string) []string {
	out := []string{}
	for _, pe := range edits.Positions {
		before := len(out)
		for _, l := range pe.Lines() {
			for _, id := range l.AchievementIDs {
				if !slices.Contains(out, id) {
					out = append(out, id)
				}
			}
		}
		if len(out) > before {
			continue
		}
		for _, p := range bank {
			if p.ID != pe.PositionID {
				continue
			}
			for _, a := range p.Achievements {
				if slices.Contains(previous, a.ID) && !slices.Contains(out, a.ID) {
					out = append(out, a.ID)
				}
			}
		}
	}
	return out
}

func (pl plan) overlay(edits cvedit.EditSet, slots []dto.SlotEdit) (cvedit.EditSet, error) {
	texts := make(map[string]string, len(slots))
	var changes []dto.SlotEdit
	for _, e := range slots {
		if e.Remove {
			changes = append(changes, e)
			continue
		}
		text := strings.TrimSpace(e.Text)
		if text == "" || strings.ContainsAny(text, "\r\n") {
			return edits, apperr.Invalid("an edit must be one non-empty line")
		}
		if e.After != "" {
			e.Text = text
			changes = append(changes, e)
			continue
		}
		texts[e.SlotID] = text
	}
	if text, ok := texts[profileSlotID]; ok {
		if pl.structure.Profile == nil {
			return edits, apperr.Invalid("unknown slot")
		}
		current := pl.structure.Profile.Text
		if edits.Profile != nil {
			current = *edits.Profile
		}
		if text != current {
			edits.Profile = &text
		}
		delete(texts, profileSlotID)
	}
	slotIDs := pl.slotIDs()
	for _, pe := range edits.Positions {
		if len(pe.Bullets) > len(slotIDs[pe.PositionID]) {
			return edits, apperr.Conflict("the CV's headings changed since this draft was made")
		}
		for i := range pe.Bullets {
			id := slotIDs[pe.PositionID][i]
			if text, ok := texts[id]; ok && text != pe.Bullets[i].Text {
				pe.Bullets[i].Text, pe.Bullets[i].Keep = text, false
			}
			delete(texts, id)
		}
	}
	if len(texts) > 0 {
		return edits, apperr.Invalid("unknown slot")
	}
	return pl.applyAddedChanges(edits, changes)
}

func (pl plan) applyAddedChanges(edits cvedit.EditSet, changes []dto.SlotEdit) (cvedit.EditSet, error) {
	slotIDs := pl.slotIDs()
	for _, c := range changes {
		if !strings.HasPrefix(c.SlotID, addedSlotPrefix) {
			if c.Remove {
				return edits, apperr.Invalid("only an added bullet can be removed")
			}
			return edits, apperr.Invalid("unknown slot")
		}
		if len(c.SlotID) > maxAddedSlotIDLen || strings.ContainsAny(c.SlotID, " \t\r\n/") {
			return edits, apperr.Invalid("invalid slot id")
		}
		if c.Remove {
			removeAdded(&edits, c.SlotID)
			continue
		}
		if err := upsertAdded(&edits, slotIDs, c); err != nil {
			return edits, err
		}
	}
	return edits, nil
}

func removeAdded(edits *cvedit.EditSet, id string) {
	for pi := range edits.Positions {
		pe := &edits.Positions[pi]
		at := slices.IndexFunc(pe.Added, func(a cvedit.AddedBullet) bool { return a.SlotID == id })
		if at < 0 {
			continue
		}
		after := pe.Added[at].After
		pe.Added = slices.Delete(pe.Added, at, at+1)
		for i := range pe.Added {
			if pe.Added[i].After == id {
				pe.Added[i].After = after
			}
		}
		return
	}
}

func upsertAdded(edits *cvedit.EditSet, slotIDs docedit.PositionSlots, c dto.SlotEdit) error {
	for pi := range edits.Positions {
		pe := &edits.Positions[pi]
		if at := slices.IndexFunc(pe.Added, func(a cvedit.AddedBullet) bool { return a.SlotID == c.SlotID }); at >= 0 {
			pe.Added[at].Text = c.Text
			return nil
		}
	}
	for pi := range edits.Positions {
		pe := &edits.Positions[pi]
		base := slotIDs[pe.PositionID]
		follows := slices.Contains(base[:min(len(pe.Bullets), len(base))], c.After) ||
			slices.ContainsFunc(pe.Added, func(a cvedit.AddedBullet) bool { return a.SlotID == c.After })
		if !follows {
			continue
		}
		pe.Added = append(pe.Added, cvedit.AddedBullet{SlotID: c.SlotID, After: c.After, Text: c.Text})
		return nil
	}
	return apperr.Invalid("unknown slot")
}

func (pl plan) overlaySkills(edits cvedit.EditSet, groups []dto.SkillGroup) (cvedit.EditSet, error) {
	if len(groups) == 0 {
		return edits, nil
	}
	if pl.structure.Skills == nil || edits.LegacySkills {
		return edits, apperr.Invalid("this draft's skills cannot be edited")
	}
	base := cvedit.SkillGroups(pl.structure.Skills)
	if len(groups) != len(base) {
		return edits, apperr.Invalid("skills must keep the CV's skill lines")
	}
	sourced := slices.Concat(cvedit.FlatSkills(base), pl.bankSkills)
	var seen []string
	out := make([]cvedit.SkillGroup, len(groups))
	for i, g := range groups {
		if strings.TrimSpace(g.Label) != base[i].Label {
			return edits, apperr.Invalid("skills must keep the CV's skill line labels")
		}
		if len(g.Items) == 0 {
			return edits, apperr.Invalid("a skill line cannot be empty")
		}
		items := make([]string, len(g.Items))
		for j, it := range g.Items {
			it = strings.TrimSpace(it)
			if it == "" || strings.ContainsAny(it, "\r\n") {
				return edits, apperr.Invalid("a skill must be one non-empty line")
			}
			at := hasItem(sourced, it)
			if at < 0 {
				return edits, apperr.Invalid(it + " is neither in the CV's skills nor a Bank Skill")
			}
			it = sourced[at]
			if hasItem(seen, it) >= 0 {
				return edits, apperr.Invalid(it + " appears twice in the skills")
			}
			seen = append(seen, it)
			items[j] = it
		}
		out[i] = cvedit.SkillGroup{Label: base[i].Label, Items: items}
	}
	edits.Skills = out
	return edits, nil
}

func skillsEqual(a, b []cvedit.SkillGroup) bool {
	return slices.EqualFunc(a, b, func(x, y cvedit.SkillGroup) bool {
		return x.Label == y.Label && slices.Equal(x.Items, y.Items)
	})
}

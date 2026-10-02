package cvtailor

import (
	"context"
	"fmt"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/docparse"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/docedit"
)

type planned struct {
	cvedit.Position
	slotIDs []string
	texts   map[string]string
}

type plan struct {
	structure docparse.DocStructure
	positions []planned
	bank      []string
}

func (s *Service) plan(ctx context.Context, claim dto.DraftClaim, docID string) (plan, error) {
	ds, err := loadTab(ctx, s.docs, claim.UserID, docID, claim.TabID)
	if err != nil {
		return plan{}, err
	}
	return s.planOf(ctx, claim, ds)
}

func (s *Service) planOf(ctx context.Context, claim dto.DraftClaim, ds docparse.DocStructure) (plan, error) {
	bank, err := s.store.ListPositions(ctx, claim.UserID)
	if err != nil {
		return plan{}, err
	}
	mappings, err := s.store.ListHeadingMappings(ctx, claim.UserID, claim.DocID, claim.TabID)
	if err != nil {
		return plan{}, err
	}
	positionOfHeading := make(map[string]string, len(mappings))
	for _, hm := range mappings {
		if hm.PositionID != nil {
			positionOfHeading[hm.HeadingText] = *hm.PositionID
		}
	}

	slotsOf := map[string][]docparse.Slot{}
	for _, slot := range ds.Slots {
		if slot.HeadingIndex < 0 {
			continue
		}
		if pid, ok := positionOfHeading[ds.Headings[slot.HeadingIndex].Text]; ok {
			slotsOf[pid] = append(slotsOf[pid], slot)
		}
	}

	confirmed := make(map[string]bool, len(claim.AchievementIDs))
	for _, id := range claim.AchievementIDs {
		confirmed[id] = true
	}
	pl := plan{structure: ds}
	found := 0
	seen := map[string]bool{}
	for _, p := range bank {
		pp := planned{
			Position: cvedit.Position{ID: p.ID, Employer: p.Employer, Title: p.Title},
			texts:    map[string]string{},
		}
		for _, a := range p.Achievements {
			pl.bank = append(pl.bank, a.Text)
			if confirmed[a.ID] {
				found++
				key := normalizeText(a.Text)
				if seen[key] {
					continue
				}
				seen[key] = true
				pp.Achievements = append(pp.Achievements, cvedit.Achievement{ID: a.ID, Text: a.Text})
				pp.texts[a.ID] = a.Text
			}
		}
		if len(pp.Achievements) == 0 {
			continue
		}
		slots := slotsOf[p.ID]
		if len(slots) == 0 {
			return plan{}, apperr.Unprocessable("a chosen position is no longer mapped to a heading of the CV tab")
		}
		for _, s := range slots {
			pp.SlotTexts = append(pp.SlotTexts, s.Text)
			pp.slotIDs = append(pp.slotIDs, s.ID)
		}
		pl.positions = append(pl.positions, pp)
	}
	if found != len(confirmed) {
		return plan{}, apperr.Unprocessable("a chosen achievement no longer exists")
	}
	return pl, nil
}

func (pl plan) input(jobDescription string) cvedit.Input {
	in := cvedit.Input{JobDescription: jobDescription}
	if pl.structure.Profile != nil {
		in.HasProfile, in.BaseProfile = true, pl.structure.Profile.Text
	}
	if pl.structure.Skills != nil {
		in.HasSkills, in.BaseSkills = true, pl.structure.Skills.Items
	}
	for _, p := range pl.positions {
		in.Positions = append(in.Positions, p.Position)
	}
	return in
}

func (pl plan) baseContent() dto.DraftContent {
	c := dto.DraftContent{Skills: []string{}, Positions: []dto.DraftPosition{}}
	if pl.structure.Profile != nil {
		c.Profile = &pl.structure.Profile.Text
	}
	if pl.structure.Skills != nil {
		c.Skills = pl.structure.Skills.Items
	}
	for _, p := range pl.positions {
		dp := dto.DraftPosition{PositionID: p.ID, Bullets: []dto.DraftBullet{}}
		for _, text := range p.SlotTexts {
			dp.Bullets = append(dp.Bullets, dto.DraftBullet{Text: text, AchievementIDs: []string{}})
		}
		c.Positions = append(c.Positions, dp)
	}
	return c
}

func (pl plan) slotIDs() docedit.PositionSlots {
	out := make(docedit.PositionSlots, len(pl.positions))
	for _, p := range pl.positions {
		out[p.ID] = p.slotIDs
	}
	return out
}

func (pl plan) validate(edits cvedit.EditSet) error {
	byID := make(map[string]planned, len(pl.positions))
	for _, p := range pl.positions {
		byID[p.ID] = p
	}
	for _, pe := range edits.Positions {
		p, ok := byID[pe.PositionID]
		if !ok {
			return fmt.Errorf("%w: unknown position %q", errInvalidEdit, pe.PositionID)
		}
		if len(pe.Bullets) > len(p.slotIDs) {
			return fmt.Errorf("%w: %d bullets for %d slots", errInvalidEdit, len(pe.Bullets), len(p.slotIDs))
		}
		for _, b := range pe.Bullets {
			if strings.TrimSpace(b.Text) == "" || (!b.Keep && len(b.AchievementIDs) == 0) {
				return fmt.Errorf("%w: bullet with no text or citation", errInvalidEdit)
			}
			for _, id := range b.AchievementIDs {
				if _, ok := p.texts[id]; !ok {
					return fmt.Errorf("%w: achievement %q is not confirmed for position %q", errInvalidEdit, id, pe.PositionID)
				}
			}
		}
	}
	return nil
}

func (pl plan) revertBlocked(edits cvedit.EditSet) (cvedit.EditSet, []string) {
	blocks := checks.Blocking(checks.Run(pl.draft(edits, 0, 0)))
	blocked := make(map[string]bool, len(blocks))
	var names []string
	for _, f := range blocks {
		names = append(names, f.Check)
		if f.Check == checks.CheckSkills {
			edits.Skills = nil
			continue
		}
		blocked[f.SlotID] = true
	}
	if pl.structure.Profile != nil && blocked[pl.structure.Profile.ID] {
		edits.Profile = nil
	}
	byID := make(map[string]planned, len(pl.positions))
	for _, p := range pl.positions {
		byID[p.ID] = p
	}
	for _, pe := range edits.Positions {
		p := byID[pe.PositionID]
		for i := range pe.Bullets {
			if blocked[p.slotIDs[i]] {
				pe.Bullets[i] = cvedit.Bullet{Keep: true, Text: p.SlotTexts[i]}
			}
		}
	}
	return edits, names
}

func (pl plan) baseText() []string {
	var out []string
	for _, h := range pl.structure.Headings {
		out = append(out, h.Text)
	}
	for _, s := range pl.structure.Slots {
		out = append(out, s.Text)
	}
	if pl.structure.Profile != nil {
		out = append(out, pl.structure.Profile.Text)
	}
	if pl.structure.Skills != nil {
		out = append(out, pl.structure.Skills.Items...)
	}
	return out
}

func (pl plan) draft(edits cvedit.EditSet, basePages, draftPages int) checks.Draft {
	d := checks.Draft{Bank: pl.bank, Skills: edits.Skills, JobSkills: edits.JobSkills, BasePages: basePages, DraftPages: draftPages}
	d.Contact = &checks.ContactInput{InBody: pl.structure.Contact.InBody, InHeaderFooter: pl.structure.Contact.InHeaderFooter}
	if pl.structure.Skills != nil {
		d.BaseSkills = pl.structure.Skills.Items
	}
	d.BaseText = pl.baseText()
	if pl.structure.Profile != nil && edits.Profile != nil {
		d.Profile = &checks.Slot{ID: pl.structure.Profile.ID, Text: *edits.Profile, BaseText: pl.structure.Profile.Text}
	}
	byID := make(map[string]planned, len(pl.positions))
	for _, p := range pl.positions {
		byID[p.ID] = p
	}
	for _, pe := range edits.Positions {
		p := byID[pe.PositionID]
		cp := checks.Position{ID: p.ID}
		for _, a := range p.Achievements {
			cp.Achievements = append(cp.Achievements, a.Text)
		}
		for i, b := range pe.Bullets {
			slot := checks.Slot{ID: p.slotIDs[i], Text: b.Text, BaseText: p.SlotTexts[i]}
			for _, id := range b.AchievementIDs {
				slot.Cited = append(slot.Cited, p.texts[id])
			}
			cp.Bullets = append(cp.Bullets, slot)
		}
		d.Positions = append(d.Positions, cp)
	}
	return d
}

func loadTab(ctx context.Context, docs DocFetcher, userID, docID, tabID string) (docparse.DocStructure, error) {
	raw, err := docs.GetDocument(ctx, userID, docID, tabID)
	if err != nil {
		return docparse.DocStructure{}, fmt.Errorf("load CV tab: %w", err)
	}
	ds, err := docparse.Parse(raw)
	if err != nil {
		return docparse.DocStructure{}, apperr.Unprocessable("could not read the CV tab")
	}
	return ds, nil
}

func (pl plan) withSlotIDs(edits cvedit.EditSet) cvedit.EditSet {
	slotIDs := pl.slotIDs()
	for _, pe := range edits.Positions {
		for i := range pe.Bullets {
			pe.Bullets[i].SlotID = slotIDs[pe.PositionID][i]
		}
	}
	return edits
}

func normalizeText(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

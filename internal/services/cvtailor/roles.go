package cvtailor

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/docparse"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/tabcopy"
)

type roleLayout struct {
	order  []string
	roleOf map[string]string
}

func (s *Service) roleLayout(ctx context.Context, claim dto.DraftClaim, ds docparse.DocStructure) (roleLayout, error) {
	mappings, err := s.store.ListHeadingMappings(ctx, claim.UserID, claim.DocID, claim.TabID)
	if err != nil {
		return roleLayout{}, err
	}
	mapped := make(map[string]string, len(mappings))
	for _, m := range mappings {
		if m.PositionID != nil {
			mapped[m.HeadingText] = *m.PositionID
		}
	}
	rl := roleLayout{roleOf: map[string]string{}}
	for _, h := range roleHeadings(ds) {
		if pid, ok := mapped[h.Text]; ok {
			rl.roleOf[h.Text] = pid
			if !slices.Contains(rl.order, pid) {
				rl.order = append(rl.order, pid)
			}
		}
	}
	return rl, nil
}

func (rl roleLayout) validate(roles []string, bank []dto.Position) error {
	if len(roles) == 0 {
		return apperr.Invalid("a CV needs at least one role")
	}
	seen := make(map[string]bool, len(roles))
	for _, id := range roles {
		if seen[id] {
			return apperr.Invalid("roles must not repeat")
		}
		seen[id] = true
		if !slices.Contains(rl.order, id) && !slices.ContainsFunc(bank, func(p dto.Position) bool { return p.ID == id }) {
			return apperr.Invalid("unknown role")
		}
	}
	return nil
}

func (s *Service) withinRoles(ctx context.Context, claim dto.DraftClaim, roles []string) (dto.DraftClaim, error) {
	if roles == nil {
		return claim, nil
	}
	bank, err := s.store.ListPositions(ctx, claim.UserID)
	if err != nil {
		return claim, err
	}
	owner := map[string]string{}
	for _, p := range bank {
		for _, a := range p.Achievements {
			owner[a.ID] = p.ID
		}
	}
	claim.AchievementIDs = slices.DeleteFunc(slices.Clone(claim.AchievementIDs), func(id string) bool {
		return !slices.Contains(roles, owner[id])
	})
	return claim, nil
}

func (s *Service) applyRoles(ctx context.Context, claim dto.DraftClaim, ds docparse.DocStructure, edits cvedit.EditSet, roles []string, bank []dto.Position) (cvedit.EditSet, roleLayout, bool, error) {
	rl, err := s.roleLayout(ctx, claim, ds)
	if err != nil {
		return edits, rl, false, err
	}
	if err := rl.validate(roles, bank); err != nil {
		return edits, rl, false, err
	}
	current := edits.Roles
	if current == nil {
		current = rl.order
	}
	if slices.Equal(current, roles) {
		return edits, rl, false, nil
	}
	edits.Roles = roles
	edits.Positions = slices.DeleteFunc(slices.Clone(edits.Positions), func(pe cvedit.PositionEdit) bool {
		return !slices.Contains(roles, pe.PositionID)
	})
	for _, id := range roles {
		if !slices.Contains(rl.order, id) && !slices.ContainsFunc(edits.Positions, func(pe cvedit.PositionEdit) bool { return pe.PositionID == id }) {
			edits.Positions = append(edits.Positions, cvedit.PositionEdit{PositionID: id})
		}
	}
	slices.SortStableFunc(edits.Positions, func(a, b cvedit.PositionEdit) int {
		return slices.Index(roles, a.PositionID) - slices.Index(roles, b.PositionID)
	})
	return edits, rl, true, nil
}

func (s *Service) rebuildDoc(ctx context.Context, claim dto.DraftClaim, draftDocID string, rl roleLayout, edits cvedit.EditSet) (docparse.DocStructure, error) {
	base, err := s.docs.GetDocument(ctx, claim.UserID, claim.DocID, claim.TabID)
	if err != nil {
		return docparse.DocStructure{}, fmt.Errorf("load CV tab: %w", err)
	}
	bank, err := s.store.ListPositions(ctx, claim.UserID)
	if err != nil {
		return docparse.DocStructure{}, err
	}
	roles := make(map[string]tabcopy.Role, len(bank))
	for _, p := range bank {
		roles[p.ID] = tabcopy.Role{Title: p.Title, Employer: p.Employer, Start: p.StartDate, End: p.EndDate}
	}
	for _, id := range edits.Roles {
		if slices.Contains(rl.order, id) {
			continue
		}
		at := slices.IndexFunc(edits.Positions, func(pe cvedit.PositionEdit) bool { return pe.PositionID == id })
		if at < 0 || len(edits.Positions[at].Lines()) == 0 {
			return docparse.DocStructure{}, apperr.Invalid("a role added to the CV needs a bullet")
		}
	}
	arranged, headings, err := tabcopy.Arrange(base, rl.roleOf, edits.Roles, roles)
	if err != nil {
		return docparse.DocStructure{}, unrebuildable(err)
	}
	var added []dto.HeadingMapping
	for id, text := range headings {
		if _, taken := rl.roleOf[text]; taken {
			return docparse.DocStructure{}, apperr.Unprocessable("the CV already has a heading like " + text)
		}
		added = append(added, dto.HeadingMapping{HeadingText: text, PositionID: &id})
	}
	if len(added) > 0 {
		if err := s.store.SaveHeadingMappings(ctx, claim.UserID, claim.DocID, claim.TabID, added); err != nil {
			return docparse.DocStructure{}, err
		}
	}
	current, err := s.docs.GetDocument(ctx, claim.UserID, draftDocID, claim.TabID)
	if err != nil {
		return docparse.DocStructure{}, fmt.Errorf("load draft doc: %w", err)
	}
	cleared, err := tabcopy.Clear(current, claim.TabID)
	if err != nil {
		return docparse.DocStructure{}, unrebuildable(err)
	}
	rebuild, err := tabcopy.Requests(arranged, claim.TabID)
	if err != nil {
		return docparse.DocStructure{}, unrebuildable(err)
	}
	ds, err := docparse.Parse(arranged)
	if err != nil {
		return docparse.DocStructure{}, fmt.Errorf("parse rearranged tab: %w", err)
	}
	kept, err := s.withinRoles(ctx, claim, edits.Roles)
	if err != nil {
		return docparse.DocStructure{}, err
	}
	pl, err := s.planOf(ctx, kept, ds)
	if err != nil {
		return docparse.DocStructure{}, err
	}
	reapply := cvedit.EditSet{Positions: edits.Positions, Profile: edits.Profile, Skills: edits.Skills}
	reqs, err := s.editRequests(pl, reapply)
	if err != nil {
		return docparse.DocStructure{}, err
	}
	if err := s.drive.BatchUpdate(ctx, claim.UserID, draftDocID, slices.Concat(cleared, rebuild, reqs)); err != nil {
		return docparse.DocStructure{}, fmt.Errorf("rebuild draft doc: %w", err)
	}
	return ds, nil
}

func unrebuildable(err error) error {
	if errors.Is(err, tabcopy.ErrUnsupported) || errors.Is(err, tabcopy.ErrAmbiguousRole) {
		return apperr.Unprocessable("this CV's layout can't be reordered: " + err.Error())
	}
	return fmt.Errorf("rebuild draft doc: %w", err)
}

package cvtailor

import (
	"context"
	"slices"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
)

func (s *Service) resolveSwaps(ctx context.Context, userID string, in dto.DraftInput) ([]dto.SkillSwap, error) {
	if len(in.SkillSwaps) == 0 {
		return nil, nil
	}
	ds, err := loadTab(ctx, s.docs, userID, in.DocID, in.TabID)
	if err != nil {
		return nil, err
	}
	if ds.Skills == nil {
		return nil, noSkillsSection()
	}
	bank, err := s.store.ListBankSkills(ctx, userID)
	if err != nil {
		return nil, err
	}
	swaps := slices.Clone(in.SkillSwaps)
	for i, sw := range swaps {
		bi := slices.IndexFunc(bank, func(b dto.BankSkill) bool { return b.ID == sw.BankSkillID })
		if bi < 0 {
			return nil, apperr.Invalid("unknown bank skill")
		}
		swaps[i].Name = bank[bi].Name
	}
	if _, err := applySkillSwaps(cvedit.SkillGroups(ds.Skills), swaps); err != nil {
		return nil, err
	}
	return swaps, nil
}

func noSkillsSection() error {
	return apperr.Invalid("the CV tab has no skills section to swap into")
}

func applySkillSwaps(lines []cvedit.SkillGroup, swaps []dto.SkillSwap) ([]cvedit.SkillGroup, error) {
	if len(swaps) == 0 {
		return lines, nil
	}
	out := make([]cvedit.SkillGroup, len(lines))
	for i, l := range lines {
		out[i] = cvedit.SkillGroup{Label: l.Label, Items: slices.Clone(l.Items)}
	}
	for _, sw := range swaps {
		if sw.Line < 0 || sw.Line >= len(out) {
			return nil, apperr.Invalid("unknown skill line")
		}
		if slices.ContainsFunc(out, func(l cvedit.SkillGroup) bool { return hasItem(l.Items, sw.Name) >= 0 }) {
			return nil, apperr.Invalid(sw.Name + " is already in the CV's skills")
		}
		at := hasItem(out[sw.Line].Items, sw.Replaces)
		if at < 0 {
			return nil, apperr.Invalid(sw.Replaces + " is not in that skill line")
		}
		out[sw.Line].Items[at] = sw.Name
	}
	return out, nil
}

func hasItem(items []string, want string) int {
	return slices.IndexFunc(items, func(it string) bool { return strings.EqualFold(strings.TrimSpace(it), strings.TrimSpace(want)) })
}

func withoutSwaps(claim dto.DraftClaim) dto.DraftClaim {
	claim.SkillSwaps = nil
	return claim
}

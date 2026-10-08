package cvtailor

import (
	"context"
	"slices"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
)

func (s *Service) validateSwaps(ctx context.Context, userID string, in dto.DraftInput) error {
	if len(in.SkillSwaps) == 0 {
		return nil
	}
	ds, err := loadTab(ctx, s.docs, userID, in.DocID, in.TabID)
	if err != nil {
		return err
	}
	if ds.Skills == nil {
		return apperr.Invalid("the CV tab has no skills section to swap into")
	}
	bank, err := s.store.ListBankSkills(ctx, userID)
	if err != nil {
		return err
	}
	_, err = applySkillSwaps(cvedit.SkillGroups(ds.Skills), in.SkillSwaps, bank)
	return err
}

// applySkillSwaps returns lines with each swap's Bank Skill in place of the
// item it replaces. A swap naming a Bank Skill outside bank, a missing line,
// an item not in that line, or a skill the CV already lists is invalid.
func applySkillSwaps(lines []cvedit.SkillGroup, swaps []dto.SkillSwap, bank []dto.BankSkill) ([]cvedit.SkillGroup, error) {
	if len(swaps) == 0 {
		return lines, nil
	}
	out := make([]cvedit.SkillGroup, len(lines))
	for i, l := range lines {
		out[i] = cvedit.SkillGroup{Label: l.Label, Items: slices.Clone(l.Items)}
	}
	for _, sw := range swaps {
		bi := slices.IndexFunc(bank, func(b dto.BankSkill) bool { return b.ID == sw.BankSkillID })
		if bi < 0 {
			return nil, apperr.Invalid("unknown bank skill")
		}
		if sw.Line < 0 || sw.Line >= len(out) {
			return nil, apperr.Invalid("unknown skill line")
		}
		name := bank[bi].Name
		if slices.ContainsFunc(out, func(l cvedit.SkillGroup) bool { return hasItem(l.Items, name) >= 0 }) {
			return nil, apperr.Invalid(name + " is already in the CV's skills")
		}
		at := hasItem(out[sw.Line].Items, sw.Replaces)
		if at < 0 {
			return nil, apperr.Invalid(sw.Replaces + " is not in that skill line")
		}
		out[sw.Line].Items[at] = name
	}
	return out, nil
}

func hasItem(items []string, want string) int {
	return slices.IndexFunc(items, func(it string) bool { return strings.EqualFold(strings.TrimSpace(it), strings.TrimSpace(want)) })
}

// withoutSwaps is the claim for planning a Draft Doc, whose Skill Lines
// already hold the swaps.
func withoutSwaps(claim dto.DraftClaim) dto.DraftClaim {
	claim.SkillSwaps = nil
	return claim
}

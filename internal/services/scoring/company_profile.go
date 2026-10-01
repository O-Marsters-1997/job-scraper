package scoring

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
)

// CompanyProfiles rolls userID's picked Options up across each Company's open
// Jobs from cached Answers only; it never calls an Answerer. Every picked
// Option appears for every Company, with Known zero when nothing is answered.
func (m *Module) CompanyProfiles(ctx context.Context, userID string, companyIDs []string) (map[string][]dto.CompanyProfileEntry, error) {
	cfg, err := m.store.GetSearchConfig(ctx, userID)
	if err != nil && !notFound(err) {
		return nil, err
	}
	options, err := m.store.ListScoringOptions(ctx)
	if err != nil {
		return nil, err
	}
	byID := newBank(options).byID
	answersByCompany, err := m.store.ListCompanyAnswers(ctx, companyIDs, jev.Model)
	if err != nil {
		return nil, err
	}

	picked := pickedOptions(cfg.Preferences.Picks, byID)
	hashes := make([]string, len(picked))
	for i, opt := range picked {
		hashes[i] = QuestionHash(opt.Question)
	}

	out := make(map[string][]dto.CompanyProfileEntry, len(companyIDs))
	for _, companyID := range companyIDs {
		jobs := answersByCompany[companyID]
		entries := make([]dto.CompanyProfileEntry, len(picked))
		for i, opt := range picked {
			entries[i] = profileEntry(opt, hashes[i], jobs)
		}
		out[companyID] = entries
	}
	return out, nil
}

func pickedOptions(picks []dto.Pick, byID map[string]dto.ScoringOption) []dto.ScoringOption {
	var picked []dto.ScoringOption
	for _, p := range dedupeBySource(picks) {
		if opt, ok := byID[p.OptionID]; ok && opt.RetiredAt == nil {
			picked = append(picked, opt)
		}
	}
	return picked
}

func profileEntry(opt dto.ScoringOption, hash string, jobs []map[string]dto.Answer) dto.CompanyProfileEntry {
	e := dto.CompanyProfileEntry{Dimension: opt.Dimension, Label: opt.Label, Total: len(jobs)}
	for _, answers := range jobs {
		a, ok := answers[hash]
		if !ok {
			continue
		}
		switch resolveAnswer(a) {
		case "yes":
			e.Known++
			e.Yes++
		case "no":
			e.Known++
		}
	}
	return e
}

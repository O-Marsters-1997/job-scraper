package cvtailor

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/profiledraft"
)

func profileRedraftChecks() []string {
	return []string{
		"Grounding: every claim must come from your experience bank or skills.",
		"Banned words: the redraft is rejected if it uses one.",
		"Line budget: the redraft may not run longer than the current profile.",
	}
}

func (m *Module) PromptLab(ctx context.Context, userID string, q dto.PromptLabQuery) (dto.PromptLab, error) {
	draft, err := m.editableDraft(ctx, userID, q.ID)
	if err != nil {
		return dto.PromptLab{}, err
	}
	profile, err := baseProfile(draft)
	if err != nil {
		return dto.PromptLab{}, err
	}
	job, err := m.store.JobBrief(ctx, draft.JobID)
	if err != nil {
		return dto.PromptLab{}, err
	}
	g := profiledraft.Generation
	lab := dto.PromptLab{Generation: string(g.Name), Main: g.Main, Checks: profileRedraftChecks()}
	for _, p := range g.Parts {
		lab.Parts = append(lab.Parts, dto.PromptPart{Path: p.Path, Text: p.Default})
	}
	for _, d := range profiledraft.Documents(profiledraft.Input{
		Profile: profile, JobTitle: job.Title, Company: job.Company, JobDescription: job.Description,
	}) {
		lab.Documents = append(lab.Documents, dto.PromptDocument{Path: d.Path, Text: d.Text})
	}
	return lab, nil
}

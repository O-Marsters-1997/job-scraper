package cvtailor

import (
	"context"
	"iter"
	"log/slog"
	"unicode/utf8"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/profiledraft"
)

const maxPartRunes = 20000

func profileRedraftChecks() []string {
	return []string{
		"Grounding: every claim must come from your experience bank or skills.",
		"Banned words: the redraft is rejected if it uses one.",
		"Line budget: the redraft may not run longer than the current profile.",
	}
}

type PromptTryEvent struct {
	Delta string
	Done  *dto.PromptTryDone
	Err   error
}

func (m *Module) PromptTry(ctx context.Context, userID string, in dto.PromptTryInput) (iter.Seq[PromptTryEvent], error) {
	known := make(map[string]bool, len(profiledraft.Generation.Parts))
	for _, p := range profiledraft.Generation.Parts {
		known[p.Path] = true
	}
	for path, text := range in.Parts {
		if !known[path] {
			return nil, apperr.Invalid("unknown prompt file " + path)
		}
		if utf8.RuneCountInString(text) > maxPartRunes {
			return nil, apperr.Invalid(path + " is too long")
		}
	}
	draft, key, input, err := m.redraftInput(ctx, userID, in.ID, in.Steer, in.Parts)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	events := make(chan PromptTryEvent, suggestBuffer)
	go m.streamTry(ctx, events, key, draft.ID, input)

	return func(yield func(PromptTryEvent) bool) {
		defer cancel()
		for ev := range events {
			if !yield(ev) {
				return
			}
		}
	}, nil
}

func (m *Module) streamTry(ctx context.Context, events chan<- PromptTryEvent, key, draftID string, in profiledraft.Input) {
	defer close(events)
	send := func(e PromptTryEvent) {
		select {
		case events <- e:
		case <-ctx.Done():
		}
	}
	res, err := m.profile.Try(ctx, key, in, func(delta string) { send(PromptTryEvent{Delta: delta}) })
	m.log.InfoContext(ctx, "profile try", slog.String("draft_id", draftID), slog.String("model", res.Model),
		slog.String("prompt_key", res.PromptKey), slog.Float64("cost", res.Cost), slog.Any(logger.KeyErr, err))
	if err != nil {
		send(PromptTryEvent{Err: suggestError(err)})
		return
	}
	send(PromptTryEvent{Done: &dto.PromptTryDone{
		Text: res.Text, Findings: toDraftFindings(res.Findings), Cost: res.Cost, PromptKey: res.PromptKey,
	}})
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

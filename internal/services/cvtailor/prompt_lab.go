package cvtailor

import (
	"context"
	"iter"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/profiledraft"
)

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
	if err := validateParts(in.Parts); err != nil {
		return nil, err
	}
	key, input, err := m.redraftInput(ctx, userID, in.ID, in.Steer, in.Parts)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	events := make(chan PromptTryEvent, suggestBuffer)
	go m.streamTry(ctx, events, key, in.ID, input)

	return func(yield func(PromptTryEvent) bool) {
		defer cancel()
		for ev := range events {
			if !yield(ev) {
				return
			}
		}
	}, nil
}

func validateParts(parts map[string]string) error {
	known := make(map[string]bool, len(profiledraft.Generation.Parts))
	for _, p := range profiledraft.Generation.Parts {
		known[p.Path] = true
	}
	for path, text := range parts {
		if !known[path] {
			return apperr.Invalid("unknown prompt file " + path)
		}
		if utf8.RuneCountInString(text) > maxPartRunes {
			return apperr.Invalid(path + " is too long")
		}
	}
	return nil
}

func (m *Module) UseProfileTry(ctx context.Context, userID string, in dto.ProfileUseInput) (dto.Draft, error) {
	text := strings.TrimSpace(in.Text)
	if err := validateParts(in.Parts); err != nil {
		return dto.Draft{}, err
	}
	_, input, err := m.redraftInput(ctx, userID, in.ID, in.Steer, in.Parts)
	if err != nil {
		return dto.Draft{}, err
	}
	res := profiledraft.Use(input, text)
	if res.Attempt.Rejection != "" {
		return dto.Draft{}, apperr.Unprocessable("this text breaks the rules: " + res.Attempt.Rejection)
	}
	saved, err := m.svc.SaveDraftSlots(ctx, userID, dto.DraftSlotsInput{
		ID: in.ID, Slots: []dto.SlotEdit{{SlotID: profileSlotID, Text: text}},
	})
	if err != nil {
		return dto.Draft{}, err
	}
	if err := m.store.AddProfileAttempts(ctx, userID, in.ID, toProfileAttempts([]profiledraft.Attempt{res.Attempt})); err != nil {
		return dto.Draft{}, err
	}
	if res.Overlong {
		saved.RedraftWarning = "The text is longer than your current profile. Shorten it to keep the page layout."
	}
	return saved, nil
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
	parts, err := m.resolvedParts(ctx, userID, g)
	if err != nil {
		return dto.PromptLab{}, err
	}
	for _, p := range parts {
		lab.Parts = append(lab.Parts, dto.PromptPart{Path: p.Path, Text: p.Text, Source: string(p.Source)})
	}
	for _, d := range profiledraft.Documents(profiledraft.Input{
		Profile: profile, JobTitle: job.Title, Company: job.Company, JobDescription: job.Description,
	}) {
		lab.Documents = append(lab.Documents, dto.PromptDocument{Path: d.Path, Text: d.Text})
	}
	return lab, nil
}

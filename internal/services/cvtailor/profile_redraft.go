package cvtailor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/docparse"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/profiledraft"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/tabcopy"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
)

const maxSteerRunes = 200

func (m *Module) redraftInput(ctx context.Context, userID, draftID, steer string, parts map[string]string) (string, profiledraft.Input, error) {
	steer = strings.TrimSpace(steer)
	if strings.ContainsAny(steer, "\r\n") || len([]rune(steer)) > maxSteerRunes {
		return "", profiledraft.Input{}, apperr.Invalid("the steer must be one line of at most 200 characters")
	}
	draft, err := m.editableDraft(ctx, userID, draftID)
	if err != nil {
		return "", profiledraft.Input{}, err
	}
	profile, err := baseProfile(draft)
	if err != nil {
		return "", profiledraft.Input{}, err
	}
	key, err := m.creds.Get(ctx, userID, jev.Provider)
	if errors.Is(err, data.ErrNotFound) {
		return "", profiledraft.Input{}, apperr.Unprocessable("connect an OpenRouter key in Settings, AI to redraft the profile")
	}
	if err != nil {
		return "", profiledraft.Input{}, fmt.Errorf("load credential: %w", err)
	}
	job, err := m.store.JobBrief(ctx, draft.JobID)
	if err != nil {
		return "", profiledraft.Input{}, err
	}
	skills, err := m.store.ListBankSkills(ctx, userID)
	if err != nil {
		return "", profiledraft.Input{}, err
	}
	positions, err := m.store.ListPositions(ctx, userID)
	if err != nil {
		return "", profiledraft.Input{}, err
	}
	input := profiledraft.Input{
		Profile: profile, Bank: indexBank(positions, nil).all, Steer: steer, Parts: parts,
		JobTitle: job.Title, Company: job.Company, JobDescription: job.Description,
	}
	for _, s := range skills {
		input.BankSkills = append(input.BankSkills, s.Name)
	}
	if input.Lines, err = m.profileLines(ctx, userID, draft); err != nil {
		return "", profiledraft.Input{}, err
	}
	return key, input, nil
}

func (m *Module) RedraftProfile(ctx context.Context, userID string, in dto.ProfileRedraftInput) (dto.Draft, error) {
	key, input, err := m.redraftInput(ctx, userID, in.ID, in.Steer, nil)
	if err != nil {
		return dto.Draft{}, err
	}

	res, redraftErr := m.profile.Redraft(ctx, key, input)
	m.log.InfoContext(ctx, "profile redraft", slog.String("draft_id", in.ID),
		slog.Int("attempts", len(res.Attempts)), slog.Bool("accepted", res.Text != ""), slog.Any(logger.KeyErr, redraftErr))
	if len(res.Attempts) > 0 {
		if err := m.store.AddProfileAttempts(ctx, userID, in.ID, toProfileAttempts(res.Attempts)); err != nil {
			return dto.Draft{}, err
		}
	}
	if redraftErr != nil {
		return dto.Draft{}, suggestError(redraftErr)
	}
	if res.Text == "" {
		reason := ""
		if n := len(res.Attempts); n > 0 {
			reason = res.Attempts[n-1].Rejection
		}
		return dto.Draft{}, apperr.Unprocessable("the redraft broke the rules: " + reason)
	}
	saved, err := m.svc.SaveDraftSlots(ctx, userID, dto.DraftSlotsInput{
		ID: in.ID, Slots: []dto.SlotEdit{{SlotID: profileSlotID, Text: res.Text}},
	})
	if err == nil && res.Overlong {
		saved.RedraftWarning = "The redraft is longer than your current profile. Shorten it to keep the page layout."
	}
	return saved, err
}

func baseProfile(draft dto.Draft) (string, error) {
	var base baseContent
	if len(draft.BaseContent) == 0 || json.Unmarshal(draft.BaseContent, &base) != nil || base.Profile == nil {
		return "", apperr.Unprocessable("this CV has no profile to redraft")
	}
	return *base.Profile, nil
}

func (m *Module) editableDraft(ctx context.Context, userID, id string) (dto.Draft, error) {
	draft, err := m.store.GetDraft(ctx, userID, id)
	if err != nil {
		return dto.Draft{}, err
	}
	if draft.Status != statusReady || draft.DraftDocID == "" {
		return dto.Draft{}, apperr.Conflict("only a ready draft can be edited")
	}
	if outcome(draft) == dto.OutcomeDiscarded {
		return dto.Draft{}, apperr.Conflict("this draft was discarded")
	}
	return draft, nil
}

func (m *Module) profileLines(ctx context.Context, userID string, draft dto.Draft) (func(string) int, error) {
	raw, err := m.docs.GetDocument(ctx, userID, draft.DraftDocID, draft.DraftTabID)
	if err != nil {
		return nil, fmt.Errorf("load draft doc: %w", err)
	}
	doc, err := tabcopy.Layout(raw)
	if err != nil {
		return nil, fmt.Errorf("lay out draft doc: %w", err)
	}
	ds, err := docparse.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse draft doc: %w", err)
	}
	if ds.Profile == nil {
		return nil, nil
	}
	for i := range doc.Blocks {
		if doc.Blocks[i].StartIndex == ds.Profile.StartIndex {
			geo, block := pageGeometry{page: doc.Page}, &doc.Blocks[i]
			return func(text string) int { return geo.lines(text, block) }, nil
		}
	}
	return nil, nil
}

func (m *Module) ProfileCaptures(ctx context.Context, userID string) ([]dto.ProfileCapture, error) {
	return m.store.ListProfileCaptures(ctx, userID)
}

func toProfileAttempts(in []profiledraft.Attempt) []dto.ProfileAttempt {
	out := make([]dto.ProfileAttempt, len(in))
	for i, a := range in {
		out[i] = dto.ProfileAttempt{Text: a.Text, Steer: a.Steer, Rejection: a.Rejection, PromptKey: a.PromptKey, Model: a.Model, Cost: a.Cost}
	}
	return out
}

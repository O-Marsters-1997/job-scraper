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
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/profiledraft"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
)

const maxSteerRunes = 200

func (m *Module) RedraftProfile(ctx context.Context, userID string, in dto.ProfileRedraftInput) (dto.Draft, error) {
	steer := strings.TrimSpace(in.Steer)
	if strings.ContainsAny(steer, "\r\n") || len([]rune(steer)) > maxSteerRunes {
		return dto.Draft{}, apperr.Invalid("the steer must be one line of at most 200 characters")
	}
	draft, err := m.store.GetDraft(ctx, userID, in.ID)
	if err != nil {
		return dto.Draft{}, err
	}
	if draft.Status != statusReady || draft.DraftDocID == "" {
		return dto.Draft{}, apperr.Conflict("only a ready draft can be edited")
	}
	if outcome(draft) == dto.OutcomeDiscarded {
		return dto.Draft{}, apperr.Conflict("this draft was discarded")
	}
	var base baseContent
	if len(draft.BaseContent) == 0 || json.Unmarshal(draft.BaseContent, &base) != nil || base.Profile == nil {
		return dto.Draft{}, apperr.Unprocessable("this CV has no profile to redraft")
	}
	key, err := m.creds.Get(ctx, userID, jev.Provider)
	if errors.Is(err, data.ErrNotFound) {
		return dto.Draft{}, apperr.Unprocessable("connect an OpenRouter key in Settings, AI to redraft the profile")
	}
	if err != nil {
		return dto.Draft{}, fmt.Errorf("load credential: %w", err)
	}
	job, err := m.store.JobBrief(ctx, draft.JobID)
	if err != nil {
		return dto.Draft{}, err
	}
	skills, err := m.store.ListBankSkills(ctx, userID)
	if err != nil {
		return dto.Draft{}, err
	}
	positions, err := m.store.ListPositions(ctx, userID)
	if err != nil {
		return dto.Draft{}, err
	}
	input := profiledraft.Input{
		Profile: *base.Profile, Bank: indexBank(positions, nil).all, Steer: steer,
		JobTitle: job.Title, Company: job.Company, JobDescription: job.Description,
	}
	for _, s := range skills {
		input.BankSkills = append(input.BankSkills, s.Name)
	}

	res, redraftErr := m.profile.Redraft(ctx, key, input)
	m.log.InfoContext(ctx, "profile redraft", slog.String("draft_id", draft.ID),
		slog.Int("attempts", len(res.Attempts)), slog.Bool("accepted", res.Text != ""), slog.Any(logger.KeyErr, redraftErr))
	if len(res.Attempts) > 0 {
		if err := m.store.AddProfileAttempts(ctx, userID, draft.ID, toProfileAttempts(res.Attempts)); err != nil {
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
	return m.svc.SaveDraftSlots(ctx, userID, dto.DraftSlotsInput{
		ID: draft.ID, Slots: []dto.SlotEdit{{SlotID: profileSlotID, Text: res.Text}},
	})
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

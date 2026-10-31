package cvtailor

import (
	"context"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/outreachhook"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/profiledraft"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
)

const (
	maxHookJobChars = 3000
	maxHookRunes    = 600
)

func (m *Module) WriteHook(ctx context.Context, userID string, in dto.HookInput) (dto.OutreachDraft, error) {
	subject, err := m.outreachSubject(ctx, userID, in.ContactID)
	if err != nil {
		return dto.OutreachDraft{}, err
	}
	key, err := m.creds.Get(ctx, userID, jev.Provider)
	if errors.Is(err, data.ErrNotFound) {
		return dto.OutreachDraft{}, apperr.Unprocessable("connect an OpenRouter key in Settings, AI to write a hook")
	}
	if err != nil {
		return dto.OutreachDraft{}, fmt.Errorf("load credential: %w", err)
	}
	chosen, err := m.chosenOutreachAchievement(ctx, userID, subject.JobID, in.AchievementID)
	if err != nil {
		return dto.OutreachDraft{}, err
	}
	task := outreachhook.Input{JobTitle: subject.JobTitle, Company: subject.Company, ContactRole: subject.Role}
	if chosen != nil {
		task.Achievement = chosen.Text
	}
	if subject.JobID != "" {
		brief, err := m.store.JobBrief(ctx, subject.JobID)
		if err != nil {
			return dto.OutreachDraft{}, err
		}
		task.JobDescription = truncateRunes(profiledraft.JobText(brief.Description), maxHookJobChars)
	}

	res, err := m.hooks.Write(ctx, key, task)
	if err != nil {
		return dto.OutreachDraft{}, suggestError(err)
	}
	if res.Text == "" || utf8.RuneCountInString(res.Text) > maxHookRunes {
		return dto.OutreachDraft{}, apperr.Upstream("the model returned an unusable hook, try again")
	}
	err = m.store.SaveOutreachHook(ctx, userID, in.ContactID, res.Text, outreachhook.Model, res.Cost)
	if errors.Is(err, store.ErrContactNotFound) {
		return dto.OutreachDraft{}, apperr.NotFound("contact not found")
	}
	if err != nil {
		return dto.OutreachDraft{}, err
	}
	return m.renderDraft(ctx, userID, dto.DraftQuery{ContactID: in.ContactID, AchievementID: in.AchievementID}, subject)
}

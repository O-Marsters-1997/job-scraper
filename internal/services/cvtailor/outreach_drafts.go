package cvtailor

import (
	"cmp"
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store"
)

const hookSlot = "{hook}"

var recruiterRole = regexp.MustCompile(`(?i)recruit|talent|consultant`)

func suggestedKind(s dto.OutreachSubject) dto.OutreachKind {
	switch {
	case s.SentSinceReply > 0:
		return dto.OutreachFollowUp
	case recruiterRole.MatchString(s.Role):
		return dto.OutreachRecruiter
	}
	return dto.OutreachHiringManager
}

// OutreachDraft renders a Contact's Outreach template server-side. It reads
// only cached Jev answers and never calls a model.
func (m *Module) OutreachDraft(ctx context.Context, userID string, q dto.DraftQuery) (dto.OutreachDraft, error) {
	subject, err := m.outreachSubject(ctx, userID, q.ContactID)
	if err != nil {
		return dto.OutreachDraft{}, err
	}
	return m.renderDraft(ctx, userID, q, subject)
}

func (m *Module) outreachSubject(ctx context.Context, userID, contactID string) (dto.OutreachSubject, error) {
	subject, err := m.store.OutreachSubject(ctx, userID, contactID)
	if errors.Is(err, store.ErrContactNotFound) {
		return dto.OutreachSubject{}, apperr.NotFound("contact not found")
	}
	return subject, err
}

func (m *Module) renderDraft(ctx context.Context, userID string, q dto.DraftQuery, subject dto.OutreachSubject) (dto.OutreachDraft, error) {
	templates, err := m.store.ListOutreachTemplates(ctx, userID)
	if err != nil {
		return dto.OutreachDraft{}, err
	}
	tpl, ok := pickTemplate(templates, q.TemplateID, suggestedKind(subject))
	if !ok {
		return dto.OutreachDraft{}, apperr.NotFound("no template to draft from")
	}
	chosen, err := m.chosenOutreachAchievement(ctx, userID, subject.JobID, q.AchievementID)
	if err != nil {
		return dto.OutreachDraft{}, err
	}
	hook, cost, err := m.store.OutreachHook(ctx, userID, q.ContactID)
	if err != nil {
		return dto.OutreachDraft{}, err
	}

	out := dto.OutreachDraft{
		TemplateID: tpl.ID, Kind: tpl.Kind, HasHookSlot: strings.Contains(tpl.Body, hookSlot),
		Hook: hook, HookFindings: []dto.LetterFinding{}, Cost: cost,
	}
	if hook != "" {
		out.HookFindings = checks.Hook(hook, subject.Company)
	}
	achievement := "[achievement]"
	if chosen != nil {
		out.AchievementID, achievement = chosen.ID, chosen.Text
	}
	out.Body = strings.NewReplacer(
		"{first_name}", firstName(subject.Name),
		"{company}", orPlaceholder(subject.Company, "company"),
		"{role}", orPlaceholder(subject.JobTitle, subject.Role, "role"),
		"{achievement}", achievement,
		hookSlot, cmp.Or(hook, "[hook]"),
	).Replace(tpl.Body)
	return out, nil
}

func (m *Module) chosenOutreachAchievement(ctx context.Context, userID, jobID, achievementID string) (*dto.Achievement, error) {
	achievements, err := m.store.ListAchievementsNewestFirst(ctx, userID)
	if err != nil {
		return nil, err
	}
	if achievementID == "" {
		return m.matchedAchievement(ctx, jobID, achievements)
	}
	for i := range achievements {
		if achievements[i].ID == achievementID {
			return &achievements[i], nil
		}
	}
	return nil, apperr.NotFound("achievement not found")
}

func pickTemplate(templates []dto.OutreachTemplate, id string, kind dto.OutreachKind) (dto.OutreachTemplate, bool) {
	for _, t := range templates {
		if id != "" && t.ID == id || id == "" && t.Kind == kind && t.IsDefault {
			return t, true
		}
	}
	return dto.OutreachTemplate{}, false
}

func (m *Module) matchedAchievement(ctx context.Context, jobID string, newestFirst []dto.Achievement) (*dto.Achievement, error) {
	if len(newestFirst) == 0 {
		return nil, nil
	}
	best := &newestFirst[0]
	if jobID == "" || m.cached == nil {
		return best, nil
	}
	questions := make([]string, len(newestFirst))
	for i, a := range newestFirst {
		questions[i] = achievementQuestion(a.Text)
	}
	answers, err := m.cached.CachedAnswers(ctx, jobID, questions)
	if err != nil {
		return nil, err
	}
	bestLean, found := 0.0, false
	for i, a := range newestFirst {
		ans, ok := answers[achievementQuestion(a.Text)]
		if lean := ans.PYes - ans.PNo; ok && (!found || lean > bestLean) {
			best, bestLean, found = &newestFirst[i], lean, true
		}
	}
	return best, nil
}

func firstName(name string) string {
	if f := strings.Fields(name); len(f) > 0 {
		return f[0]
	}
	return "there"
}

// orPlaceholder returns the first non-empty value, else a visible [label].
func orPlaceholder(parts ...string) string {
	label := parts[len(parts)-1]
	for _, p := range parts[:len(parts)-1] {
		if p != "" {
			return p
		}
	}
	return "[" + label + "]"
}

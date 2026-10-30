package applications

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func tooLong(field, v string, limit int) error {
	if utf8.RuneCountInString(v) > limit {
		return apperr.Invalid(fmt.Sprintf("%s must be at most %d characters", field, limit))
	}
	return nil
}

func (s *Service) CreateContact(ctx context.Context, userID string, in dto.ContactInput) (dto.Contact, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return dto.Contact{}, apperr.Invalid("name is required")
	}
	for _, f := range []struct {
		field, value string
		limit        int
	}{
		{"name", in.Name, 200}, {"role", in.Role, 200}, {"company_name", in.CompanyName, 200},
		{"linkedin_url", in.LinkedInURL, 2048}, {"email", in.Email, 320},
	} {
		if err := tooLong(f.field, f.value, f.limit); err != nil {
			return dto.Contact{}, err
		}
	}
	if in.LinkedInURL != "" && !strings.HasPrefix(in.LinkedInURL, "http://") && !strings.HasPrefix(in.LinkedInURL, "https://") {
		return dto.Contact{}, apperr.Invalid("linkedin_url must start with http:// or https://")
	}
	return s.store.CreateContact(ctx, userID, in, time.Now())
}

func (s *Service) MarkContactSent(ctx context.Context, userID string, in dto.MarkSentInput) (dto.ContactSent, error) {
	switch in.Channel {
	case dto.ContactChannelLinkedIn, dto.ContactChannelEmail, dto.ContactChannelCall, dto.ContactChannelInbound:
	default:
		return dto.ContactSent{}, apperr.Invalid("invalid channel")
	}
	switch in.TemplateKind {
	case "", dto.OutreachRecruiter, dto.OutreachHiringManager, dto.OutreachReferral, dto.OutreachFollowUp:
	default:
		return dto.ContactSent{}, apperr.Invalid("invalid template_kind")
	}
	if err := tooLong("template_name", in.TemplateName, 100); err != nil {
		return dto.ContactSent{}, err
	}
	return s.store.MarkContactSent(ctx, userID, in, time.Now())
}

func (s *Service) RecordContactReply(ctx context.Context, userID string, in dto.RecordReplyInput) (dto.Contact, error) {
	if !slices.Contains([]dto.ContactOutcome{dto.ContactOutcomeInterested, dto.ContactOutcomeReferred, dto.ContactOutcomeNotNow, dto.ContactOutcomeNoThanks}, in.Outcome) {
		return dto.Contact{}, apperr.Invalid("invalid outcome")
	}
	if err := tooLong("reply", in.Reply, 4000); err != nil {
		return dto.Contact{}, err
	}
	return s.store.RecordContactReply(ctx, userID, in, time.Now())
}

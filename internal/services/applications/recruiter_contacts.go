package applications

import (
	"context"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func (s *Service) LogRecruiterContact(ctx context.Context, userID string, in dto.LogRecruiterContactInput) (dto.RecruiterContact, error) {
	if _, err := time.Parse(time.DateOnly, in.ContactedOn); err != nil {
		return dto.RecruiterContact{}, apperr.Invalid("invalid contacted_on date")
	}
	switch in.Channel {
	case dto.ChannelLinkedInMessage, dto.ChannelEmail, dto.ChannelCall, dto.ChannelInbound:
	default:
		return dto.RecruiterContact{}, apperr.Invalid("invalid channel")
	}
	switch in.Outcome {
	case dto.OutcomeNoReply, dto.OutcomeRejected, dto.OutcomeSpoke, dto.OutcomeSubmitted:
	default:
		return dto.RecruiterContact{}, apperr.Invalid("invalid outcome")
	}
	return s.store.CreateRecruiterContact(ctx, userID, in)
}

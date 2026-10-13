package applications

import (
	"context"
	"slices"
	"strings"
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

// RecruiterContactSummaries returns each of q.JobIDs's contact summary, keyed by job ID.
func (s *Service) RecruiterContactSummaries(ctx context.Context, userID string, q dto.RecruiterContactSummariesQuery) (map[string]dto.RecruiterContactSummary, error) {
	var ids []string
	for _, id := range strings.Split(q.JobIDs, ",") {
		if id = strings.TrimSpace(id); id != "" && !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return map[string]dto.RecruiterContactSummary{}, nil
	}
	return s.store.GetRecruiterContactSummaries(ctx, userID, ids)
}

package applications

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	maxBody         = 10000
	maxQuoteExact   = 2000
	maxQuoteContext = 64
)

func validateBody(kind, body string) error {
	if n := utf8.RuneCountInString(body); n < 1 || n > maxBody {
		return apperr.Invalid(fmt.Sprintf("%s must be 1 to %d characters", kind, maxBody))
	}
	return nil
}

func (s *Service) CreateJobNote(ctx context.Context, userID string, in dto.CreateJobNoteInput) (dto.JobNote, error) {
	if err := validateBody("note", in.Body); err != nil {
		return dto.JobNote{}, err
	}
	return s.store.CreateJobNote(ctx, userID, in)
}

func (s *Service) UpdateJobNote(ctx context.Context, userID string, in dto.UpdateJobNoteInput) (dto.JobNote, error) {
	if err := validateBody("note", in.Body); err != nil {
		return dto.JobNote{}, err
	}
	return s.store.UpdateJobNote(ctx, userID, in)
}

func (s *Service) CreateJobComment(ctx context.Context, userID string, in dto.CreateJobCommentInput) (dto.JobComment, error) {
	if err := validateBody("comment", in.Body); err != nil {
		return dto.JobComment{}, err
	}
	q := in.Quote
	if n := utf8.RuneCountInString(q.Exact); n < 1 || n > maxQuoteExact {
		return dto.JobComment{}, apperr.Invalid(fmt.Sprintf("quote must be 1 to %d characters", maxQuoteExact))
	}
	if utf8.RuneCountInString(q.Prefix) > maxQuoteContext || utf8.RuneCountInString(q.Suffix) > maxQuoteContext {
		return dto.JobComment{}, apperr.Invalid(fmt.Sprintf("quote context must be at most %d characters", maxQuoteContext))
	}
	return s.store.CreateJobComment(ctx, userID, in)
}

func (s *Service) UpdateJobComment(ctx context.Context, userID string, in dto.UpdateJobCommentInput) (dto.JobComment, error) {
	if err := validateBody("comment", in.Body); err != nil {
		return dto.JobComment{}, err
	}
	return s.store.UpdateJobComment(ctx, userID, in)
}

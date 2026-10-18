package applications

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	maxJobNoteBody  = 10000
	maxQuoteExact   = 2000
	maxQuoteContext = 64
)

func validateJobNoteBody(body string) error {
	if n := utf8.RuneCountInString(body); n < 1 || n > maxJobNoteBody {
		return apperr.Invalid(fmt.Sprintf("note must be 1 to %d characters", maxJobNoteBody))
	}
	return nil
}

func (s *Service) CreateJobNote(ctx context.Context, userID string, in dto.CreateJobNoteInput) (dto.JobNote, error) {
	if err := validateJobNoteBody(in.Body); err != nil {
		return dto.JobNote{}, err
	}
	if q := in.Quote; q != nil {
		if n := utf8.RuneCountInString(q.Exact); n < 1 || n > maxQuoteExact {
			return dto.JobNote{}, apperr.Invalid(fmt.Sprintf("quote must be 1 to %d characters", maxQuoteExact))
		}
		if utf8.RuneCountInString(q.Prefix) > maxQuoteContext || utf8.RuneCountInString(q.Suffix) > maxQuoteContext {
			return dto.JobNote{}, apperr.Invalid(fmt.Sprintf("quote context must be at most %d characters", maxQuoteContext))
		}
	}
	return s.store.CreateJobNote(ctx, userID, in)
}

func (s *Service) UpdateJobNote(ctx context.Context, userID string, in dto.UpdateJobNoteInput) (dto.JobNote, error) {
	if err := validateJobNoteBody(in.Body); err != nil {
		return dto.JobNote{}, err
	}
	return s.store.UpdateJobNote(ctx, userID, in)
}

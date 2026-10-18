package applications

import (
	"context"
	"unicode/utf8"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

const maxJobNoteBody = 10000

func (s *Service) CreateJobNote(ctx context.Context, userID string, in dto.CreateJobNoteInput) (dto.JobNote, error) {
	if n := utf8.RuneCountInString(in.Body); n < 1 || n > maxJobNoteBody {
		return dto.JobNote{}, apperr.Invalid("note must be 1 to 10000 characters")
	}
	return s.store.CreateJobNote(ctx, userID, in)
}

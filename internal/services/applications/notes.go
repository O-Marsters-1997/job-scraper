package applications

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

const maxJobNoteBody = 10000

func (s *Service) CreateJobNote(ctx context.Context, userID string, in dto.CreateJobNoteInput) (dto.JobNote, error) {
	if n := utf8.RuneCountInString(in.Body); n < 1 || n > maxJobNoteBody {
		return dto.JobNote{}, apperr.Invalid(fmt.Sprintf("note must be 1 to %d characters", maxJobNoteBody))
	}
	return s.store.CreateJobNote(ctx, userID, in)
}

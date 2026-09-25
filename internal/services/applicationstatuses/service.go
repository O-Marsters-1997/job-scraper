package applicationstatuses

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

type Service struct {
	statuses providers.ApplicationStatusProvider
}

func New(statuses providers.ApplicationStatusProvider) *Service {
	return &Service{statuses: statuses}
}

func (s *Service) Create(ctx context.Context, userID string, in dto.ApplicationStatusInput) (dto.ApplicationStatus, error) {
	if in.Name == "" || in.Colour == "" {
		return dto.ApplicationStatus{}, apperr.Invalid("name and colour are required")
	}
	return s.statuses.CreateApplicationStatus(ctx, userID, in.Name, in.Colour)
}

func (s *Service) Update(ctx context.Context, userID string, in dto.ApplicationStatusInput) (dto.ApplicationStatus, error) {
	if in.Name == "" || in.Colour == "" {
		return dto.ApplicationStatus{}, apperr.Invalid("name and colour are required")
	}
	return s.statuses.UpdateApplicationStatus(ctx, in.ID, userID, in.Name, in.Colour)
}

// Delete refuses to delete a status still in use, reporting how many
// applications use it.
func (s *Service) Delete(ctx context.Context, userID, id string) error {
	count, err := s.statuses.CountApplicationsUsingStatus(ctx, id, userID)
	if err != nil {
		return err
	}
	if count > 0 {
		return apperr.WithFields(apperr.Conflict("status is in use"), map[string]any{"count": count})
	}
	return s.statuses.DeleteApplicationStatus(ctx, id, userID)
}

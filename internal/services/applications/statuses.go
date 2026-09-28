package applications

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func (s *Service) CreateStatus(ctx context.Context, userID string, in dto.ApplicationStatusInput) (dto.ApplicationStatus, error) {
	if in.Name == "" || in.Colour == "" {
		return dto.ApplicationStatus{}, apperr.Invalid("name and colour are required")
	}
	return s.store.CreateApplicationStatus(ctx, userID, in.Name, in.Colour)
}

func (s *Service) UpdateStatus(ctx context.Context, userID string, in dto.ApplicationStatusInput) (dto.ApplicationStatus, error) {
	if in.Name == "" || in.Colour == "" {
		return dto.ApplicationStatus{}, apperr.Invalid("name and colour are required")
	}
	return s.store.UpdateApplicationStatus(ctx, in.ID, userID, in.Name, in.Colour)
}

func (s *Service) DeleteStatus(ctx context.Context, userID, id string) error {
	count, err := s.store.CountApplicationsUsingStatus(ctx, id, userID)
	if err != nil {
		return err
	}
	if count > 0 {
		return apperr.WithFields(apperr.Conflict("status is in use"), map[string]any{"count": count})
	}
	return s.store.DeleteApplicationStatus(ctx, id, userID)
}

// Package applicationstatuses validates and orchestrates CRUD on a user's
// application Statuses.
package applicationstatuses

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

type Store interface {
	CreateApplicationStatus(ctx context.Context, userID, name, colour string) (dto.ApplicationStatus, error)
	UpdateApplicationStatus(ctx context.Context, id, userID, name, colour string) (dto.ApplicationStatus, error)
	DeleteApplicationStatus(ctx context.Context, id, userID string) error
	CountApplicationsUsingStatus(ctx context.Context, statusID, userID string) (int64, error)
}

type Service struct {
	store Store
}

func New(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, userID string, in dto.ApplicationStatusInput) (dto.ApplicationStatus, error) {
	if in.Name == "" || in.Colour == "" {
		return dto.ApplicationStatus{}, apperr.Invalid("name and colour are required")
	}
	return s.store.CreateApplicationStatus(ctx, userID, in.Name, in.Colour)
}

func (s *Service) Update(ctx context.Context, userID string, in dto.ApplicationStatusInput) (dto.ApplicationStatus, error) {
	if in.Name == "" || in.Colour == "" {
		return dto.ApplicationStatus{}, apperr.Invalid("name and colour are required")
	}
	return s.store.UpdateApplicationStatus(ctx, in.ID, userID, in.Name, in.Colour)
}

// Delete refuses to delete a status still in use, reporting how many
// applications use it.
func (s *Service) Delete(ctx context.Context, userID, id string) error {
	count, err := s.store.CountApplicationsUsingStatus(ctx, id, userID)
	if err != nil {
		return err
	}
	if count > 0 {
		return apperr.WithFields(apperr.Conflict("status is in use"), map[string]any{"count": count})
	}
	return s.store.DeleteApplicationStatus(ctx, id, userID)
}

// Package applications holds the domain rules for creating and updating a
// job application. Persistence goes through providers.ApplicationProvider.
package applications

import (
	"context"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/fp"
)

type Service struct {
	applications providers.ApplicationProvider
}

func New(applications providers.ApplicationProvider) *Service {
	return &Service{applications: applications}
}

func (s *Service) Create(ctx context.Context, userID string, in dto.CreateApplicationInput) (dto.Application, error) {
	if in.JobID == "" {
		return dto.Application{}, apperr.Invalid("job_id is required")
	}
	if !validAppliedAt(in.AppliedAt) {
		return dto.Application{}, apperr.Invalid("invalid applied_at date")
	}
	return s.applications.CreateApplication(ctx, userID, in)
}

func (s *Service) Update(ctx context.Context, userID, id string, in dto.UpdateApplicationInput) (dto.Application, error) {
	if !validAppliedAt(in.AppliedAt) {
		return dto.Application{}, apperr.Invalid("invalid applied_at date")
	}
	return s.applications.UpdateApplication(ctx, userID, id, in)
}

func validAppliedAt(date fp.Option[string]) bool {
	if date.IsNone() {
		return true
	}
	_, err := time.Parse(time.DateOnly, date.Unwrap())
	return err == nil
}

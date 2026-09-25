// Package applications holds the domain rules for creating and updating a
// job application. Persistence goes through providers.ApplicationProvider.
package applications

import (
	"context"
	"strings"
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

func (s *Service) Update(ctx context.Context, userID string, in dto.UpdateApplicationInput) (dto.Application, error) {
	if !validAppliedAt(in.AppliedAt) {
		return dto.Application{}, apperr.Invalid("invalid applied_at date")
	}
	return s.applications.UpdateApplication(ctx, userID, in.ID, in)
}

func (s *Service) Delete(ctx context.Context, userID, id string) error {
	return s.applications.DeleteApplication(ctx, userID, id)
}

// List returns the caller's applications, optionally filtered to one status.
func (s *Service) List(ctx context.Context, userID string, q dto.ApplicationsQuery) ([]dto.ApplicationWithDetails, error) {
	if q.StatusID != "" {
		return s.applications.ListApplicationsByUserAndStatus(ctx, userID, q.StatusID)
	}
	return s.applications.ListApplicationsByUser(ctx, userID)
}

// ForJobs returns each of q.JobIDs's application summary, keyed by job ID. An
// empty JobIDs returns an empty map rather than looking anything up.
func (s *Service) ForJobs(ctx context.Context, userID string, q dto.ApplicationsForJobsQuery) (map[string]dto.JobApplicationSummary, error) {
	if q.JobIDs == "" {
		return map[string]dto.JobApplicationSummary{}, nil
	}
	return s.applications.GetApplicationsForJobs(ctx, userID, strings.Split(q.JobIDs, ","))
}

func validAppliedAt(date fp.Option[string]) bool {
	if date.IsNone() {
		return true
	}
	_, err := time.Parse(time.DateOnly, date.Unwrap())
	return err == nil
}

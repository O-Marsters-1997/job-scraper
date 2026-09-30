package applications

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

type Store interface {
	CreateApplication(ctx context.Context, userID string, in dto.CreateApplicationInput) (dto.Application, error)
	ListApplications(ctx context.Context, userID, statusID string) ([]dto.ApplicationWithDetails, error)
	UpdateApplication(ctx context.Context, userID, id string, in dto.UpdateApplicationInput) (dto.Application, error)
	DeleteApplication(ctx context.Context, userID, id string) error
	GetApplicationsForJobs(ctx context.Context, userID string, jobIDs []string) (map[string]dto.JobApplicationSummary, error)

	CreateApplicationStatus(ctx context.Context, userID, name, colour string) (dto.ApplicationStatus, error)
	UpdateApplicationStatus(ctx context.Context, id, userID, name, colour string) (dto.ApplicationStatus, error)
	DeleteApplicationStatus(ctx context.Context, id, userID string) error
	CountApplicationsUsingStatus(ctx context.Context, statusID, userID string) (int64, error)
	ListApplicationStatusesByUser(ctx context.Context, userID string) ([]dto.ApplicationStatus, error)
	SeedDefaultStatuses(ctx context.Context, tx pgx.Tx, userID string) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, userID string, in dto.CreateApplicationInput) (dto.Application, error) {
	if in.JobID == "" {
		return dto.Application{}, apperr.Invalid("job_id is required")
	}
	if !validAppliedAt(in.AppliedAt) {
		return dto.Application{}, apperr.Invalid("invalid applied_at date")
	}
	return s.store.CreateApplication(ctx, userID, in)
}

func (s *Service) Update(ctx context.Context, userID string, in dto.UpdateApplicationInput) (dto.Application, error) {
	if !validAppliedAt(in.AppliedAt) {
		return dto.Application{}, apperr.Invalid("invalid applied_at date")
	}
	return s.store.UpdateApplication(ctx, userID, in.ID, in)
}

func (s *Service) Delete(ctx context.Context, userID, id string) error {
	return s.store.DeleteApplication(ctx, userID, id)
}

func (s *Service) List(ctx context.Context, userID string, q dto.ApplicationsQuery) ([]dto.ApplicationWithDetails, error) {
	return s.store.ListApplications(ctx, userID, q.StatusID)
}

// ForJobs returns each of q.JobIDs's application summary, keyed by job ID.
func (s *Service) ForJobs(ctx context.Context, userID string, q dto.ApplicationsForJobsQuery) (map[string]dto.JobApplicationSummary, error) {
	if q.JobIDs == "" {
		return map[string]dto.JobApplicationSummary{}, nil
	}
	return s.store.GetApplicationsForJobs(ctx, userID, strings.Split(q.JobIDs, ","))
}

func validAppliedAt(date *string) bool {
	if date == nil {
		return true
	}
	_, err := time.Parse(time.DateOnly, *date)
	return err == nil
}

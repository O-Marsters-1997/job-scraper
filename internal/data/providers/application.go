package providers

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

var (
	ErrApplicationExists = apperr.Conflict("application already exists for this job")
	ErrNotFound          = apperr.NotFound("not found")
)

type ApplicationStatusProvider interface {
	SeedDefaultStatuses(ctx context.Context, userID string) error
	CreateApplicationStatus(ctx context.Context, userID, name, colour string) (dto.ApplicationStatus, error)
	ListApplicationStatusesByUser(ctx context.Context, userID string) ([]dto.ApplicationStatus, error)
	UpdateApplicationStatus(ctx context.Context, id, userID, name, colour string) (dto.ApplicationStatus, error)
	DeleteApplicationStatus(ctx context.Context, id, userID string) error
	CountApplicationsUsingStatus(ctx context.Context, statusID, userID string) (int64, error)
}

type ApplicationProvider interface {
	CreateApplication(ctx context.Context, userID string, in dto.CreateApplicationInput) (dto.Application, error)
	ListApplicationsByUser(ctx context.Context, userID string) ([]dto.ApplicationWithDetails, error)
	ListApplicationsByUserAndStatus(ctx context.Context, userID, statusID string) ([]dto.ApplicationWithDetails, error)
	UpdateApplication(ctx context.Context, userID, id string, in dto.UpdateApplicationInput) (dto.Application, error)
	DeleteApplication(ctx context.Context, userID, id string) error
	GetApplicationsForJobs(ctx context.Context, userID string, jobIDs []string) (map[string]dto.JobApplicationSummary, error)
}

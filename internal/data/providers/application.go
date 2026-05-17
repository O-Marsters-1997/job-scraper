package providers

import (
	"context"
	"errors"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

var (
	ErrApplicationExists = errors.New("application already exists for this job")
	ErrNotFound          = errors.New("not found")
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
	CreateApplication(ctx context.Context, userID, jobID, statusID, notes, salaryInfo string, appliedAt *string) (dto.Application, error)
	ListApplicationsByUser(ctx context.Context, userID string) ([]dto.ApplicationWithDetails, error)
	ListApplicationsByUserAndStatus(ctx context.Context, userID, statusID string) ([]dto.ApplicationWithDetails, error)
	UpdateApplication(ctx context.Context, id, userID, statusID, notes, salaryInfo string, appliedAt *string) (dto.Application, error)
	DeleteApplication(ctx context.Context, id, userID string) error
	GetApplicationsForJobs(ctx context.Context, userID string, jobIDs []string) (map[string]dto.JobApplicationSummary, error)
}

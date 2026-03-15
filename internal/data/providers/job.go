package providers

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type JobProvider interface {
	UpsertJobs(ctx context.Context, jobs []dto.Job) error
	UpsertJob(ctx context.Context, job dto.Job) error
}

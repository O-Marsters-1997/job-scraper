package providers

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/sources"
)

type JobProvider interface {
	UpsertJobs(ctx context.Context, jobs []sources.Job) error
	UpsertJob(ctx context.Context, job sources.Job) error
}

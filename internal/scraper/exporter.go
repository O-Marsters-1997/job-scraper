package scraper

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type JobExporter interface {
	BulkExport(ctx context.Context, jobs []dto.Job) error
}

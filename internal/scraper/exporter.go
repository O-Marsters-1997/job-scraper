package scraper

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// JobExporter is the egress port both source paths terminate at.
// Production uses APIExporter; tests use an in-memory recorder.
type JobExporter interface {
	BulkExport(ctx context.Context, jobs []dto.Job) error
}

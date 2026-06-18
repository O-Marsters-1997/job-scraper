package scraper

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// JobPublisher is the egress port both source paths terminate at.
// Production uses APIPublisher; tests use an in-memory recorder.
type JobPublisher interface {
	Publish(ctx context.Context, jobs []dto.Job) error
}

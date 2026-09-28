package jobsearchtest

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
)

// NoopScoring satisfies jobsearch.ScoringPort with no-op writes and a
// primed Search Config, in place of a real scoring.Module.
type NoopScoring struct {
	configs map[string]dto.SearchConfig
}

func NewNoopScoring() *NoopScoring {
	return &NoopScoring{configs: make(map[string]dto.SearchConfig)}
}

func (s *NoopScoring) SeedSearchConfig(cfg dto.SearchConfig) {
	s.configs[cfg.UserID] = cfg
}

func (s *NoopScoring) SearchConfig(_ context.Context, userID string) (dto.SearchConfig, error) {
	return s.configs[userID], nil
}

func (s *NoopScoring) JobsChanged(context.Context, pgx.Tx, []string, bool) error { return nil }
func (s *NoopScoring) JobsClosed(context.Context, pgx.Tx, []string) error        { return nil }
func (s *NoopScoring) CompanyTracked(context.Context, pgx.Tx, string, string) error {
	return nil
}

var _ jobsearch.ScoringPort = (*NoopScoring)(nil)

// NoopQueue satisfies jobsearch.QueuePublisher without a real broker.
type NoopQueue struct{}

func (NoopQueue) Publish(context.Context, queue.Task) error          { return nil }
func (NoopQueue) EnqueueJobs(context.Context, []dto.QueuedJob) error { return nil }

var _ jobsearch.QueuePublisher = NoopQueue{}

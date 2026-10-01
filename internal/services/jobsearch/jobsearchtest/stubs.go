package jobsearchtest

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
)

type NoopScoring struct {
	configs  map[string]dto.SearchConfig
	profiles map[string][]dto.CompanyProfileEntry
}

func NewNoopScoring() *NoopScoring {
	return &NoopScoring{configs: make(map[string]dto.SearchConfig), profiles: make(map[string][]dto.CompanyProfileEntry)}
}

func (s *NoopScoring) SeedSearchConfig(cfg dto.SearchConfig) {
	s.configs[cfg.UserID] = cfg
}

func (s *NoopScoring) SearchConfig(_ context.Context, userID string) (dto.SearchConfig, error) {
	return s.configs[userID], nil
}

func (s *NoopScoring) CompanyProfiles(_ context.Context, _ string, companyIDs []string) (map[string][]dto.CompanyProfileEntry, error) {
	out := make(map[string][]dto.CompanyProfileEntry, len(companyIDs))
	for _, id := range companyIDs {
		out[id] = s.profiles[id]
	}
	return out, nil
}

func (s *NoopScoring) SeedProfile(companyID string, profile []dto.CompanyProfileEntry) {
	s.profiles[companyID] = profile
}

func (s *NoopScoring) JobsChanged(context.Context, pgx.Tx, []string, bool) error { return nil }
func (s *NoopScoring) JobsClosed(context.Context, pgx.Tx, []string) error        { return nil }
func (s *NoopScoring) CompanyTracked(context.Context, pgx.Tx, string, string) error {
	return nil
}

var _ jobsearch.ScoringPort = (*NoopScoring)(nil)

type NoopQueue struct{}

func (NoopQueue) Publish(context.Context, queue.Task) error          { return nil }
func (NoopQueue) EnqueueJobs(context.Context, []dto.QueuedJob) error { return nil }

var _ jobsearch.QueuePublisher = NoopQueue{}

// Package jobsearch is the jobsearch context: jobs, companies, company
// boards, source targets, candidates and ingest (ADR 0011).
package jobsearch

import (
	"context"
	"errors"
	"strconv"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
	"github.com/ollymarsters/job-scraper/internal/services/sourcetargets"
	"github.com/ollymarsters/job-scraper/internal/slug"
)

const (
	defaultPageLimit  = 50
	maxSeenJobIDs     = 5000
	maxBulkCompanyIDs = 500
)

type Service struct {
	store   Store
	queue   QueuePublisher
	configs sourcetargets.SearchConfigReader
}

func NewService(store Store, q QueuePublisher, configs sourcetargets.SearchConfigReader) *Service {
	return &Service{store: store, queue: q, configs: configs}
}

func (s *Service) excludedCompanySlugs(ctx context.Context, userID string) ([]string, error) {
	cfg, err := s.configs.SearchConfig(ctx, userID)
	if err != nil && !errors.Is(err, data.ErrNotFound) {
		return nil, err
	}
	slugs := make([]string, 0, len(cfg.ExcludedCompanies))
	for _, name := range cfg.ExcludedCompanies {
		if s := slug.Company(name); s != "" {
			slugs = append(slugs, s)
		}
	}
	return slugs, nil
}

func (s *Service) Get(ctx context.Context, userID, id string) (dto.Job, error) {
	job, err := s.store.GetJob(ctx, id, userID)
	switch {
	case errors.Is(err, data.ErrNotFound):
		return dto.Job{}, apperr.NotFound("job not found")
	case errors.Is(err, store.ErrInvalidID):
		return dto.Job{}, apperr.Invalid("invalid job ID")
	case err != nil:
		return dto.Job{}, err
	}
	if job.Listings, err = s.store.ListJobListings(ctx, job.ID); err != nil {
		return dto.Job{}, err
	}
	return job, nil
}

func (s *Service) MarkSeen(ctx context.Context, userID string, in dto.SeenInput) (struct{}, error) {
	if len(in.JobIDs) > maxSeenJobIDs {
		return struct{}{}, apperr.Invalid("too many job IDs")
	}
	err := s.store.MarkJobsSeen(ctx, userID, in.JobIDs, in.Seen)
	if errors.Is(err, store.ErrInvalidID) {
		return struct{}{}, apperr.Invalid("invalid job ID")
	}
	return struct{}{}, err
}

func parsePageLimit(raw string) (int, error) {
	if raw == "" {
		return defaultPageLimit, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 || limit > 100 {
		return 0, apperr.Invalid("limit must be between 1 and 100")
	}
	return limit, nil
}

package candidates

import (
	"context"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/score"
)

const batchSize = 100

type Candidate struct {
	ID   string
	URL  string
	Card dto.Job
}

type Store interface {
	SaveCards(context.Context, dto.SourceTarget, []dto.Job) ([]Candidate, error)
	ListForUser(context.Context, string, string, int) ([]Candidate, error)
	Assess(context.Context, string, string, time.Time, bool) (bool, error)
	ReleaseDetail(context.Context, string) error
}

type JobQueue interface {
	EnqueueJobs(context.Context, []dto.QueuedJob) error
}

type Service struct {
	store Store
	queue JobQueue
}

func New(store Store, queue JobQueue) *Service {
	return &Service{store: store, queue: queue}
}

func (s *Service) CapturePage(ctx context.Context, target dto.SourceTarget, cards []dto.Job, config dto.SearchConfig) error {
	candidates, err := s.store.SaveCards(ctx, target, cards)
	if err != nil {
		return fmt.Errorf("save candidate cards: %w", err)
	}
	return s.assess(ctx, candidates, config)
}

func (s *Service) Reconsider(ctx context.Context, config dto.SearchConfig) error {
	afterID := ""
	for {
		batch, err := s.store.ListForUser(ctx, config.UserID, afterID, batchSize)
		if err != nil {
			return fmt.Errorf("list candidates: %w", err)
		}
		if len(batch) == 0 {
			return nil
		}
		if err := s.assess(ctx, batch, config); err != nil {
			return err
		}
		afterID = batch[len(batch)-1].ID
		if len(batch) < batchSize {
			return nil
		}
	}
}

func (s *Service) assess(ctx context.Context, candidates []Candidate, config dto.SearchConfig) error {
	for _, candidate := range candidates {
		_, rejected := score.Reject(candidate.Card, config)
		queueDetail, err := s.store.Assess(ctx, candidate.ID, config.UserID, config.UpdatedAt, !rejected)
		if err != nil {
			return fmt.Errorf("assess candidate %s: %w", candidate.ID, err)
		}
		if !queueDetail {
			continue
		}
		if err := s.queue.EnqueueJobs(ctx, []dto.QueuedJob{{URL: candidate.URL, Card: candidate.Card}}); err != nil {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			releaseErr := s.store.ReleaseDetail(cleanupCtx, candidate.ID)
			cancel()
			if releaseErr != nil {
				return fmt.Errorf("enqueue candidate %s: %w (release: %v)", candidate.ID, err, releaseErr)
			}
			return fmt.Errorf("enqueue candidate %s: %w", candidate.ID, err)
		}
	}
	return nil
}

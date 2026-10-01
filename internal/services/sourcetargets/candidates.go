package sourcetargets

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/filter"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
)

const batchSize = 100

type Candidate struct {
	ID   string
	URL  string
	Card dto.Job
}

type CandidateStore interface {
	SaveCards(context.Context, dto.SourceTarget, []dto.Job) ([]Candidate, error)
	ListForUser(context.Context, string, string, int) ([]Candidate, error)
	Assess(context.Context, string, string, time.Time, bool) (bool, error)
	MarkDetailPending(context.Context, string) error
	DeleteExpiredCandidates(context.Context) error
}

// CardBoards reads the verified Boards of the Companies that cards name.
type CardBoards interface {
	VerifiedBoardsBySlug(ctx context.Context, slugs []string) ([]dto.CardBoard, error)
}

func (s *Service) CapturePage(ctx context.Context, target dto.SourceTarget, cards []dto.Job, config dto.SearchConfig) error {
	cards, err := s.dropBoardCards(ctx, target, cards)
	if err != nil {
		return err
	}
	candidates, err := s.targets.SaveCards(ctx, target, cards)
	if err != nil {
		return fmt.Errorf("save candidate cards: %w", err)
	}
	return s.assess(ctx, candidates, config)
}

func (s *Service) dropBoardCards(ctx context.Context, target dto.SourceTarget, cards []dto.Job) ([]dto.Job, error) {
	if target.Source != "linkedin" || len(cards) == 0 {
		return cards, nil
	}
	slugs := make([]string, len(cards))
	for i, card := range cards {
		slugs[i] = card.CompanySlug
	}
	boards, err := s.boards.VerifiedBoardsBySlug(ctx, slugs)
	if err != nil {
		return nil, fmt.Errorf("look up verified boards: %w", err)
	}
	s.harvestUntracked(ctx, boards)
	return slices.DeleteFunc(slices.Clone(cards), func(card dto.Job) bool {
		return slices.ContainsFunc(boards, func(b dto.CardBoard) bool { return b.CompanySlug == card.CompanySlug })
	}), nil
}

func (s *Service) harvestUntracked(ctx context.Context, boards []dto.CardBoard) {
	published := map[dto.CardBoard]bool{}
	for _, b := range boards {
		if b.Tracked || published[b] {
			continue
		}
		published[b] = true
		task := queue.Task{Version: 1, ID: uuid.NewString(), Source: b.Source, Kind: queue.BoardDiscoverTask, BoardToken: b.BoardToken}
		if err := s.queue.Publish(ctx, task); err != nil {
			slog.WarnContext(ctx, "could not publish board harvest",
				slog.String(logger.KeySource, b.Source), slog.String(logger.KeyCompanySlug, b.CompanySlug), slog.Any(logger.KeyErr, err))
		}
	}
}

func (s *Service) Reconsider(ctx context.Context, config dto.SearchConfig) error {
	afterID := ""
	for {
		batch, err := s.targets.ListForUser(ctx, config.UserID, afterID, batchSize)
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

func (s *Service) DeleteExpired(ctx context.Context) error {
	return s.targets.DeleteExpiredCandidates(ctx)
}

func (s *Service) assess(ctx context.Context, candidates []Candidate, config dto.SearchConfig) error {
	for _, candidate := range candidates {
		_, rejected := filter.Reject(candidate.Card, config)
		queueDetail, err := s.targets.Assess(ctx, candidate.ID, config.UserID, config.UpdatedAt, !rejected)
		if err != nil {
			return fmt.Errorf("assess candidate %s: %w", candidate.ID, err)
		}
		if !queueDetail {
			continue
		}
		if err := s.queue.EnqueueJobs(ctx, []dto.QueuedJob{{URL: candidate.URL, Card: candidate.Card}}); err != nil {
			return fmt.Errorf("enqueue candidate %s: %w", candidate.ID, err)
		}
		if err := s.targets.MarkDetailPending(ctx, candidate.ID); err != nil {
			return fmt.Errorf("mark candidate %s pending: %w", candidate.ID, err)
		}
	}
	return nil
}

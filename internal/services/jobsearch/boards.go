package jobsearch

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
)

// PolledCompanySlugs returns those of slugs whose Company has a verified Board
// that at least one User tracks.
func (s *Service) PolledCompanySlugs(ctx context.Context, slugs []string) ([]string, error) {
	return s.store.ListPolledCompanySlugs(ctx, slugs)
}

// PublishBoardChecks queues a check per active Board when manual, else per due
// Board. A failed publish is logged and skipped; only the list error returns.
func (s *Service) PublishBoardChecks(ctx context.Context, manual bool) error {
	list := s.store.ListDueBoards
	if manual {
		list = s.store.ListActiveBoards
	}
	boards, err := list(ctx)
	if err != nil {
		return err
	}
	for _, board := range boards {
		task := queue.Task{Version: 1, ID: uuid.NewString(), Source: board.Source, Kind: queue.BoardCheckTask, BoardID: board.ID, Manual: manual}
		if err := s.queue.Publish(ctx, task); err != nil {
			slog.ErrorContext(ctx, "publish Board check failed", slog.String(logger.KeyBoardID, board.ID), slog.Any(logger.KeyErr, err))
		}
	}
	return nil
}

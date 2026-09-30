package jobsearch

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
)

// RecoverRuns republishes every stuck Scrape Run. Per-target failures are
// logged; only failing to list the targets is returned.
func (m *Module) RecoverRuns(ctx context.Context) error {
	targets, err := m.sourceTargets.ListRecoverableSourceTargets(ctx)
	if err != nil {
		return err
	}
	for _, target := range targets {
		target, err = m.sourceTargets.ClaimRecoverableSourceTarget(ctx, target.ID, target.RunID)
		if errors.Is(err, data.ErrNotFound) {
			continue
		}
		if err != nil {
			slog.ErrorContext(ctx, "claim recoverable run failed", slog.String(logger.KeyTargetID, target.ID), slog.Any(logger.KeyErr, err))
			continue
		}
		task := queue.Task{Version: 1, ID: uuid.NewString(), Source: target.Source, TargetID: target.ID, RunID: target.RunID, Recovery: true}
		if role, _ := sourcespec.SourceRole(target.Source); role == sourcespec.RoleATS {
			boardID, err := m.store.GetVerifiedBoardID(ctx, target.Source, target.Value)
			if err != nil {
				slog.ErrorContext(ctx, "recover Board run failed", slog.String(logger.KeyTargetID, target.ID), slog.Any(logger.KeyErr, err))
				continue
			}
			task.Kind, task.BoardID, task.Manual = queue.BoardCheckTask, boardID, true
		} else {
			task.Kind = queue.ListingPageTask
		}
		if err := m.queue.Publish(ctx, task); err != nil {
			slog.ErrorContext(ctx, "recover run publish failed", slog.String(logger.KeyTargetID, target.ID), slog.Any(logger.KeyErr, err))
		}
	}
	return nil
}

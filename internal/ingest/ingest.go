package ingest

import (
	"context"
	"log/slog"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// Saver persists a batch of jobs.
type Saver interface {
	Save(ctx context.Context, jobs []dto.Job) error
}

// Scorer runs suitability scoring after a job is saved.
// Implementations handle their own errors internally.
type Scorer interface {
	ScoreAndSave(ctx context.Context, job dto.Job)
}

// Notifier sends a notification for a newly ingested job.
// Implementations handle their own errors internally.
type Notifier interface {
	NotifyNewJob(ctx context.Context, job dto.Job)
}

// Ingester is the ingest seam: validate → Save → score → notify.
// scorer and notifier are optional (nil skips that step).
// Save errors are returned; scoring and notification failures are handled
// internally by those implementations and never block ingest.
type Ingester struct {
	db       Saver
	scorer   Scorer
	notifier Notifier
}

// New constructs an Ingester. scorer and notifier may be nil.
// Callers must pass a nil interface value (not a nil concrete pointer) to
// avoid the typed-nil pitfall — use a declared interface variable set
// conditionally rather than casting a nil pointer directly.
func New(db Saver, scorer Scorer, notifier Notifier) *Ingester {
	return &Ingester{db: db, scorer: scorer, notifier: notifier}
}

// Ingest validates jobs (non-empty Title and URL), batch-saves the valid ones,
// then calls scorer and notifier per job. Returns the Save error if it fails;
// scoring and notify failures are fire-and-forget.
func (i *Ingester) Ingest(ctx context.Context, jobs []dto.Job) error {
	valid := make([]dto.Job, 0, len(jobs))
	for _, j := range jobs {
		if j.Title == "" || j.URL == "" {
			slog.Warn("skipping invalid job", slog.String("title", j.Title), slog.String("url", j.URL))
			continue
		}
		valid = append(valid, j)
	}
	if len(valid) == 0 {
		return nil
	}

	if err := i.db.Save(ctx, valid); err != nil {
		return err
	}
	slog.Info("jobs ingested", slog.Int("count", len(valid)))

	for _, j := range valid {
		if i.scorer != nil {
			i.scorer.ScoreAndSave(ctx, j)
		}
		if i.notifier != nil {
			i.notifier.NotifyNewJob(ctx, j)
		}
	}
	return nil
}

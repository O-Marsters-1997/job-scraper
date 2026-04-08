package notify

import (
	"context"
	"log/slog"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

type Config struct {
	To              string
	OnIngestEnabled bool
	DigestEnabled   bool
}

type NotificationService struct {
	notifier Notifier
	notifDB  providers.NotificationProvider
	jobDB    providers.JobProvider
	renderer *Renderer
	cfg      Config
}

func NewNotificationService(
	notifier Notifier,
	notifDB providers.NotificationProvider,
	jobDB providers.JobProvider,
	renderer *Renderer,
	cfg Config,
) *NotificationService {
	return &NotificationService{
		notifier: notifier,
		notifDB:  notifDB,
		jobDB:    jobDB,
		renderer: renderer,
		cfg:      cfg,
	}
}

// NotifyNewJob sends a single-job notification after a successful ingest.
// No-ops if NOTIFY_ON_INGEST is disabled.
func (s *NotificationService) NotifyNewJob(ctx context.Context, job dto.Job) {
	if !s.cfg.OnIngestEnabled {
		return
	}
	html, err := s.renderer.RenderIndividual(job)
	if err != nil {
		slog.Error("notify: render individual failed", slog.Any("err", err))
		return
	}
	if err := s.notifier.Send(ctx, s.cfg.To, "New job: "+job.Title, html); err != nil {
		slog.Error("notify: send individual failed", slog.String("title", job.Title), slog.Any("err", err))
	}
}

// SendDigest queries new jobs since the last digest, renders and sends the digest email,
// then records the send. No-ops if digest is disabled or there are no new jobs.
func (s *NotificationService) SendDigest(ctx context.Context) {
	if !s.cfg.DigestEnabled {
		return
	}

	lastSent, err := s.notifDB.GetLastDigestSentAt(ctx)
	if err != nil {
		slog.Error("notify: get last digest time failed", slog.Any("err", err))
		return
	}

	// If no digest has been sent, query all jobs.
	var jobs []dto.Job
	if lastSent.IsZero() {
		jobs, err = s.jobDB.List(ctx)
	} else {
		jobs, err = s.jobDB.ListSince(ctx, lastSent)
	}
	if err != nil {
		slog.Error("notify: list jobs for digest failed", slog.Any("err", err))
		return
	}

	if len(jobs) == 0 {
		slog.Info("notify: no new jobs since last digest, skipping")
		return
	}

	html, err := s.renderer.RenderDigest(jobs)
	if err != nil {
		slog.Error("notify: render digest failed", slog.Any("err", err))
		return
	}

	if err := s.notifier.Send(ctx, s.cfg.To, "Job digest", html); err != nil {
		slog.Error("notify: send digest failed", slog.Any("err", err))
		return
	}

	sentAt := time.Now()
	if err := s.notifDB.RecordDigest(ctx, sentAt, len(jobs)); err != nil {
		slog.Error("notify: record digest failed", slog.Any("err", err))
	}
}

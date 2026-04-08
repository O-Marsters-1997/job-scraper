package notify

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type Config struct {
	To              string
	OnIngestEnabled bool
	DigestEnabled   bool
}

type NotificationService struct {
	notifier Notifier
	renderer *Renderer
	cfg      Config
}

func NewNotificationService(
	notifier Notifier,
	renderer *Renderer,
	cfg Config,
) *NotificationService {
	return &NotificationService{
		notifier: notifier,
		renderer: renderer,
		cfg:      cfg,
	}
}

func (s *NotificationService) NotifyNewJob(ctx context.Context, job dto.Job) {
	if !s.cfg.OnIngestEnabled {
		return
	}
	log := slog.With(slog.String("title", job.Title))
	html, err := s.renderer.RenderIndividual(job)
	if err != nil {
		log.Error("render individual failed", slog.Any("err", err))
		return
	}
	if err := s.notifier.Send(ctx, s.cfg.To, "New job: "+job.Title, html); err != nil {
		log.Error("send individual failed", slog.Any("err", err))
	}
}

func (s *NotificationService) SendDigest(ctx context.Context, jobs []dto.Job) error {
	if !s.cfg.DigestEnabled {
		return nil
	}
	html, err := s.renderer.RenderDigest(jobs)
	if err != nil {
		return fmt.Errorf("notify: render digest: %w", err)
	}
	return s.notifier.Send(ctx, s.cfg.To, "Job digest", html)
}

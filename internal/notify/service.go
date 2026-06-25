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
	NotifyThreshold int // 0 = no gate; > 0 requires suitability_score >= threshold
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

func (s *NotificationService) NotifyNewJob(ctx context.Context, recipient string, job dto.Job, suitabilityScore int) {
	if !s.cfg.OnIngestEnabled {
		return
	}
	if s.cfg.NotifyThreshold > 0 && suitabilityScore < s.cfg.NotifyThreshold {
		slog.Debug("notify: skipping job below threshold",
			slog.String("title", job.Title),
			slog.Int("score", suitabilityScore),
			slog.Int("threshold", s.cfg.NotifyThreshold),
		)
		return
	}
	log := slog.With(slog.String("title", job.Title))
	html, err := s.renderer.RenderIndividual(job)
	if err != nil {
		log.Error("render individual failed", slog.Any("err", err))
		return
	}
	if err := s.notifier.Send(ctx, recipient, "New job: "+job.Title, html); err != nil {
		log.Error("send individual failed", slog.Any("err", err))
	}
}

// Threshold returns the configured notification threshold (0 = no gate).
func (s *NotificationService) Threshold() int {
	return s.cfg.NotifyThreshold
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

package notify

import (
	"context"
	"fmt"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type NotificationService struct {
	notifier Notifier
	renderer *Renderer
}

func NewNotificationService(
	notifier Notifier,
	renderer *Renderer,
) *NotificationService {
	return &NotificationService{
		notifier: notifier,
		renderer: renderer,
	}
}

func (s *NotificationService) NotifyNewJob(ctx context.Context, job dto.Job, recipientEmail string) error {
	if recipientEmail == "" {
		return nil
	}
	html, err := s.renderer.RenderIndividual(job)
	if err != nil {
		return fmt.Errorf("notify: render individual: %w", err)
	}
	return s.notifier.Send(ctx, recipientEmail, "New job: "+job.Title, html)
}

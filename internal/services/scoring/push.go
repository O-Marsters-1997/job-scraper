package scoring

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/notify"
)

func (s *Service) VAPIDPublicKey(context.Context, string) (dto.VAPIDKey, error) {
	if s.vapidKey == "" {
		return dto.VAPIDKey{}, apperr.NotFound("push is not configured")
	}
	return dto.VAPIDKey{Key: s.vapidKey}, nil
}

func (s *Service) Subscribe(ctx context.Context, userID string, sub dto.PushSubscriptionInput) (struct{}, error) {
	if sub.Endpoint == "" || sub.Keys.P256dh == "" || sub.Keys.Auth == "" {
		return struct{}{}, apperr.Invalid("endpoint and keys are required")
	}
	if err := s.store.UpsertPushSubscription(ctx, userID, sub); err != nil {
		return struct{}{}, fmt.Errorf("scoring.Subscribe: %w", err)
	}
	return struct{}{}, nil
}

func (s *Service) Unsubscribe(ctx context.Context, userID string, in dto.PushEndpointInput) (struct{}, error) {
	if err := s.store.DeletePushSubscription(ctx, userID, in.Endpoint); err != nil {
		return struct{}{}, fmt.Errorf("scoring.Unsubscribe: %w", err)
	}
	return struct{}{}, nil
}

func (s *Service) TestPush(ctx context.Context, userID string, _ struct{}) (struct{}, error) {
	err := s.pushToUser(ctx, userID, dto.PushMessage{
		Title: "FastTrack alerts are on", Body: "This is a test notification.", URL: "/", Tag: "test",
	})
	return struct{}{}, err
}

func (s *Service) pushToUser(ctx context.Context, userID string, msg dto.PushMessage) error {
	subs, err := s.store.ListPushSubscriptions(ctx, userID)
	if err != nil {
		return fmt.Errorf("scoring.pushToUser: list: %w", err)
	}
	for _, sub := range subs {
		err := s.pusher.Send(ctx, sub, msg)
		switch {
		case err == nil:
		case errors.Is(err, notify.ErrSubscriptionGone):
			if err := s.store.DeletePushSubscription(ctx, userID, sub.Endpoint); err != nil {
				slog.ErrorContext(ctx, "push subscription cleanup failed", slog.String(logger.KeyUserID, userID), slog.Any(logger.KeyErr, err))
			}
		default:
			slog.ErrorContext(ctx, "push send failed", slog.String(logger.KeyUserID, userID), slog.Any(logger.KeyErr, err))
		}
	}
	return nil
}

const maxPushLabels = 3

func newJobPush(job dto.Job, sc dto.JobScore) dto.PushMessage {
	var labels []string
	for _, row := range sc.Rows {
		if len(labels) == maxPushLabels {
			break
		}
		if row.Effect == "meets" {
			labels = append(labels, row.Label)
		}
	}
	return dto.PushMessage{
		Title: fmt.Sprintf("%d · %s, %s", sc.Score, job.Title, job.CompanySlug),
		Body:  strings.Join(labels, " · "),
		URL:   "/jobs/" + job.ID + "?from=alert",
		Tag:   job.ID,
	}
}

package store

import (
	"context"
	"fmt"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/store/sqlc"
)

func (s *Store) UpsertPushSubscription(ctx context.Context, userID string, sub dto.PushSubscriptionInput) error {
	uid, err := data.UUID(userID)
	if err != nil {
		return err
	}
	err = s.queries.UpsertPushSubscription(ctx, sqlc.UpsertPushSubscriptionParams{
		UserID: uid, Endpoint: sub.Endpoint, P256dh: sub.Keys.P256dh, Auth: sub.Keys.Auth,
	})
	if err != nil {
		return fmt.Errorf("store.UpsertPushSubscription: %w", err)
	}
	return nil
}

func (s *Store) ListPushSubscriptions(ctx context.Context, userID string) ([]dto.PushSubscriptionInput, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListPushSubscriptions(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListPushSubscriptions: %w", err)
	}
	subs := make([]dto.PushSubscriptionInput, len(rows))
	for i, r := range rows {
		subs[i] = dto.PushSubscriptionInput{Endpoint: r.Endpoint, Keys: dto.PushKeys{P256dh: r.P256dh, Auth: r.Auth}}
	}
	return subs, nil
}

func (s *Store) DeletePushSubscription(ctx context.Context, userID, endpoint string) error {
	uid, err := data.UUID(userID)
	if err != nil {
		return err
	}
	if err := s.queries.DeletePushSubscription(ctx, sqlc.DeletePushSubscriptionParams{Endpoint: endpoint, UserID: uid}); err != nil {
		return fmt.Errorf("store.DeletePushSubscription: %w", err)
	}
	return nil
}

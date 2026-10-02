package notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/SherClockHolmes/webpush-go"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// ErrSubscriptionGone means the push service says the subscription no longer
// exists (HTTP 404 or 410); the caller should delete it.
var ErrSubscriptionGone = errors.New("push subscription gone")

type WebPusher struct {
	publicKey, privateKey, subject string
}

func NewWebPusher(publicKey, privateKey, subject string) *WebPusher {
	return &WebPusher{publicKey: publicKey, privateKey: privateKey, subject: subject}
}

func (p *WebPusher) Send(ctx context.Context, sub dto.PushSubscriptionInput, msg dto.PushMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("webpush marshal: %w", err)
	}
	resp, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys:     webpush.Keys{P256dh: sub.Keys.P256dh, Auth: sub.Keys.Auth},
	}, &webpush.Options{
		Subscriber:      p.subject,
		VAPIDPublicKey:  p.publicKey,
		VAPIDPrivateKey: p.privateKey,
		TTL:             3600,
	})
	if err != nil {
		return fmt.Errorf("webpush send: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		return ErrSubscriptionGone
	case resp.StatusCode >= 300:
		return fmt.Errorf("webpush send: status %d", resp.StatusCode)
	}
	return nil
}

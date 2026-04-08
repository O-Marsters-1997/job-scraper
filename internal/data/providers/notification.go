package providers

import (
	"context"
	"time"
)

// NotificationProvider manages notification state persistence.
type NotificationProvider interface {
	// RecordDigest records that a digest email was sent.
	RecordDigest(ctx context.Context, sentAt time.Time, jobCount int) error

	// GetLastDigestSentAt returns when the last digest was sent.
	// Returns a zero time.Time if no digest has been sent yet.
	GetLastDigestSentAt(ctx context.Context) (time.Time, error)
}

package providers

import (
	"context"
	"time"
)

type NotificationProvider interface {
	RecordDigest(ctx context.Context, sentAt time.Time, jobCount int) error
	GetLastDigestSentAt(ctx context.Context) (time.Time, error)
}

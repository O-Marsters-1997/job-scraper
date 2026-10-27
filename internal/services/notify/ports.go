package notify

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// Store is notify's Postgres surface.
type Store interface {
	InsertNotification(ctx context.Context, userID, jobID, title, body string) (string, error)
}

// PhonePusher delivers one Notification to the phone; msg.URL is the Notification link.
type PhonePusher interface {
	Send(ctx context.Context, msg dto.PushMessage) error
}

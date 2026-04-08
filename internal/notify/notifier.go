package notify

import "context"

// Notifier sends a notification to a recipient.
type Notifier interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}

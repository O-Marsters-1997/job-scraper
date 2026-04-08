package notify

import "context"

type Notifier interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}

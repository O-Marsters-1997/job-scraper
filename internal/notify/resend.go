package notify

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v3"
)

type ResendNotifier struct {
	client *resend.Client
	from   string
}

func NewResendNotifier(apiKey, from string) *ResendNotifier {
	return &ResendNotifier{
		client: resend.NewClient(apiKey),
		from:   from,
	}
}

func (n *ResendNotifier) Send(ctx context.Context, to, subject, htmlBody string) error {
	_, err := n.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    n.from,
		To:      []string{to},
		Subject: subject,
		Html:    htmlBody,
	})
	if err != nil {
		return fmt.Errorf("resend send: %w", err)
	}
	return nil
}

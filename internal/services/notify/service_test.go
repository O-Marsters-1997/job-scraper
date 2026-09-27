package notify_test

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/notify"
)

type sentEmail struct{ to string }
type mockNotifier struct{ sent []sentEmail }

func (m *mockNotifier) Send(_ context.Context, to, _, _ string) error {
	m.sent = append(m.sent, sentEmail{to: to})
	return nil
}

func TestNotifyNewJobUsesRecipient(t *testing.T) {
	renderer, err := notify.NewRenderer()
	if err != nil {
		t.Fatal(err)
	}
	sender := &mockNotifier{}
	svc := notify.NewNotificationService(sender, renderer)
	job := dto.Job{ID: "job-1", Title: "Engineer", URL: "https://example.com/job"}
	for _, email := range []string{"alice@example.com", "bob@example.com", ""} {
		if err := svc.NotifyNewJob(context.Background(), job, email); err != nil {
			t.Fatal(err)
		}
	}
	if len(sender.sent) != 2 || sender.sent[0].to != "alice@example.com" || sender.sent[1].to != "bob@example.com" {
		t.Fatalf("recipients: %+v", sender.sent)
	}
}

package notify_test

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/notify"
)

func TestNotifyNewJobUsesRecipient(t *testing.T) {
	renderer, err := notify.NewRenderer()
	if err != nil {
		t.Fatal(err)
	}
	sender := &notify.MockNotifier{}
	svc := notify.NewNotificationService(sender, renderer)
	job := dto.Job{ID: "job-1", Title: "Engineer", URL: "https://example.com/job"}
	for _, email := range []string{"alice@example.com", "bob@example.com", ""} {
		if err := svc.NotifyNewJob(context.Background(), job, email); err != nil {
			t.Fatal(err)
		}
	}
	if len(sender.Sent) != 2 || sender.Sent[0].To != "alice@example.com" || sender.Sent[1].To != "bob@example.com" {
		t.Fatalf("recipients: %+v", sender.Sent)
	}
}

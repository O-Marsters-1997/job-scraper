package notify_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/notify"
)

type sentEmail struct{ To string }
type fakeNotifier struct{ sent []sentEmail }

func (n *fakeNotifier) Send(_ context.Context, to, _, _ string) error {
	n.sent = append(n.sent, sentEmail{To: to})
	return nil
}

func TestNotifyNewJobUsesRecipient(t *testing.T) {
	renderer, err := notify.NewRenderer()
	if err != nil {
		t.Fatal(err)
	}
	sender := &fakeNotifier{}
	svc := notify.NewNotificationService(sender, renderer)
	job := dto.Job{ID: "job-1", Title: "Engineer", URL: "https://example.com/job"}
	for _, email := range []string{"alice@example.com", "bob@example.com", ""} {
		if err := svc.NotifyNewJob(context.Background(), job, email); err != nil {
			t.Fatal(err)
		}
	}
	want := []sentEmail{{To: "alice@example.com"}, {To: "bob@example.com"}}
	if diff := cmp.Diff(want, sender.sent); diff != "" {
		t.Errorf("recipients mismatch, empty email skipped (-want +got):\n%s", diff)
	}
}

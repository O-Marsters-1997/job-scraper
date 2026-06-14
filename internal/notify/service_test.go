package notify_test

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/notify"
)

// stubNotifier records whether Send was called.
type stubNotifier struct {
	called  bool
	subject string
}

func (s *stubNotifier) Send(_ context.Context, _, subject, _ string) error {
	s.called = true
	s.subject = subject
	return nil
}

func newService(t *testing.T, cfg notify.Config) (*notify.NotificationService, *stubNotifier) {
	t.Helper()
	n := &stubNotifier{}
	r, err := notify.NewRenderer()
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	return notify.NewNotificationService(n, r, cfg), n
}

var testJob = dto.Job{ID: "job-1", Title: "Engineer", URL: "https://example.com/job"}

func TestNotifyNewJob_belowThreshold_doesNotSend(t *testing.T) {
	svc, n := newService(t, notify.Config{
		To:              "test@example.com",
		OnIngestEnabled: true,
		NotifyThreshold: 70,
	})

	svc.NotifyNewJob(context.Background(), testJob, 60)

	if n.called {
		t.Error("expected Send not to be called when score < threshold")
	}
}

func TestNotifyNewJob_atThreshold_sends(t *testing.T) {
	svc, n := newService(t, notify.Config{
		To:              "test@example.com",
		OnIngestEnabled: true,
		NotifyThreshold: 70,
	})

	svc.NotifyNewJob(context.Background(), testJob, 70)

	if !n.called {
		t.Error("expected Send to be called when score == threshold")
	}
}

func TestNotifyNewJob_aboveThreshold_sends(t *testing.T) {
	svc, n := newService(t, notify.Config{
		To:              "test@example.com",
		OnIngestEnabled: true,
		NotifyThreshold: 70,
	})

	svc.NotifyNewJob(context.Background(), testJob, 90)

	if !n.called {
		t.Error("expected Send to be called when score > threshold")
	}
}

func TestNotifyNewJob_zeroThreshold_alwaysSends(t *testing.T) {
	svc, n := newService(t, notify.Config{
		To:              "test@example.com",
		OnIngestEnabled: true,
		NotifyThreshold: 0,
	})

	svc.NotifyNewJob(context.Background(), testJob, 0)

	if !n.called {
		t.Error("expected Send to be called when threshold is 0")
	}
}

func TestNotifyNewJob_ingestDisabled_doesNotSend(t *testing.T) {
	svc, n := newService(t, notify.Config{
		To:              "test@example.com",
		OnIngestEnabled: false,
		NotifyThreshold: 0,
	})

	svc.NotifyNewJob(context.Background(), testJob, 100)

	if n.called {
		t.Error("expected Send not to be called when OnIngestEnabled is false")
	}
}

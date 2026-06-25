package notify_test

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/notify"
)

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

func TestNotifyNewJob(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfg      notify.Config
		score    int
		wantSent bool
	}{
		{
			name:     "below threshold does not send",
			cfg:      notify.Config{To: "test@example.com", OnIngestEnabled: true, NotifyThreshold: 70},
			score:    60,
			wantSent: false,
		},
		{
			name:     "at threshold sends",
			cfg:      notify.Config{To: "test@example.com", OnIngestEnabled: true, NotifyThreshold: 70},
			score:    70,
			wantSent: true,
		},
		{
			name:     "above threshold sends",
			cfg:      notify.Config{To: "test@example.com", OnIngestEnabled: true, NotifyThreshold: 70},
			score:    90,
			wantSent: true,
		},
		{
			name:     "zero threshold always sends",
			cfg:      notify.Config{To: "test@example.com", OnIngestEnabled: true, NotifyThreshold: 0},
			score:    0,
			wantSent: true,
		},
		{
			name:     "ingest disabled does not send",
			cfg:      notify.Config{To: "test@example.com", OnIngestEnabled: false, NotifyThreshold: 0},
			score:    100,
			wantSent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, n := newService(t, tt.cfg)
			svc.NotifyNewJob(context.Background(), testJob, tt.score, "test@example.com")
			if n.called != tt.wantSent {
				t.Errorf("Send called = %v; want %v", n.called, tt.wantSent)
			}
		})
	}
}

func TestSendDigest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfg      notify.Config
		jobs     []dto.Job
		wantSent bool
		wantErr  bool
	}{
		{
			name:     "digest disabled does not send",
			cfg:      notify.Config{To: "test@example.com", DigestEnabled: false},
			jobs:     []dto.Job{testJob},
			wantSent: false,
		},
		{
			name:     "digest enabled sends",
			cfg:      notify.Config{To: "test@example.com", DigestEnabled: true},
			jobs:     []dto.Job{testJob},
			wantSent: true,
		},
		{
			name:     "digest with empty job list sends",
			cfg:      notify.Config{To: "test@example.com", DigestEnabled: true},
			jobs:     []dto.Job{},
			wantSent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, n := newService(t, tt.cfg)
			err := svc.SendDigest(context.Background(), tt.jobs)
			if tt.wantErr && err == nil {
				t.Error("SendDigest: expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("SendDigest: unexpected error: %v", err)
			}
			if n.called != tt.wantSent {
				t.Errorf("Send called = %v; want %v", n.called, tt.wantSent)
			}
		})
	}
}

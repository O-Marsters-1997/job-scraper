package schedule_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/schedule"
)

func TestEvery(t *testing.T) {
	var logs bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	calls := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		schedule.Every(ctx, "chore", time.Millisecond, func(context.Context) error {
			calls <- struct{}{}
			return errors.New("boom")
		})
	}()

	for range 3 {
		select {
		case <-calls:
		case <-time.After(5 * time.Second):
			t.Fatal("fn was not called again")
		}
	}
	cancel()
	go func() {
		for range calls {
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Every did not return after ctx cancel")
	}
	if got := logs.String(); !strings.Contains(got, "chore failed") || !strings.Contains(got, "boom") {
		t.Errorf("log = %q, want it to name chore and error", got)
	}
}

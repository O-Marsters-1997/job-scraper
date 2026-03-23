package worker

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/queue"
)

func instantWorker() worker {
	return worker{
		itemDelay:  func() time.Duration { return 0 },
		emptyDelay: 0,
		errDelay:   0,
	}
}

func TestRun_HappyPath(t *testing.T) {
	q := queue.NewMockQueue()
	_ = q.Enqueue(context.Background(), []string{"https://example.com/job/1", "https://example.com/job/2"})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var handled []string
	handler := func(ctx context.Context, url string) error {
		handled = append(handled, url)
		if len(handled) == 2 {
			cancel() // queue is now empty; cancel so the worker exits cleanly
		}
		return nil
	}

	w := instantWorker()
	_ = w.run(ctx, q, handler)

	want := []string{"https://example.com/job/1", "https://example.com/job/2"}
	if !slices.Equal(handled, want) {
		t.Errorf("handled = %v, want %v", handled, want)
	}
}

func TestRun_EmptyQueue(t *testing.T) {
	q := queue.NewMockQueue()
	called := 0
	handler := func(ctx context.Context, url string) error {
		called++
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately so the empty-queue sleep is skipped

	w := instantWorker()
	_ = w.run(ctx, q, handler)

	if called != 0 {
		t.Errorf("handler called %d times on empty queue, want 0", called)
	}
}

func TestRun_ContextCancellation(t *testing.T) {
	q := queue.NewMockQueue()
	_ = q.Enqueue(context.Background(), []string{"https://example.com/job/1"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel

	w := instantWorker()
	err := w.run(ctx, q, func(ctx context.Context, url string) error { return nil })
	if err != nil {
		t.Errorf("run returned %v, want nil", err)
	}
}

func TestRun_HandlerErrorContinues(t *testing.T) {
	q := queue.NewMockQueue()
	_ = q.Enqueue(context.Background(), []string{
		"https://example.com/job/1",
		"https://example.com/job/2",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var handled []string
	handler := func(ctx context.Context, url string) error {
		handled = append(handled, url)
		if len(handled) == 2 {
			cancel()
		}
		return errors.New("handler error") // non-fatal; worker should continue
	}

	w := instantWorker()
	_ = w.run(ctx, q, handler)

	if len(handled) != 2 {
		t.Errorf("handled %d URLs, want 2", len(handled))
	}
}

func TestRun_DequeueError(t *testing.T) {
	q := queue.NewMockQueue()
	q.DequeueErr = errors.New("valkey unavailable")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel so the errDelay sleep exits immediately

	w := instantWorker()
	err := w.run(ctx, q, func(ctx context.Context, url string) error { return nil })
	if err != nil {
		t.Errorf("run returned %v, want nil", err)
	}
}

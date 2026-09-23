package worker

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
)

func instantWorker() worker {
	return worker{
		itemDelay:  func() time.Duration { return 0 },
		emptyDelay: 0,
		errDelay:   0,
	}
}

func enqueueURLs(t *testing.T, q *queue.MockQueue, urls []string) {
	t.Helper()
	jobs := make([]dto.QueuedJob, len(urls))
	for i, u := range urls {
		jobs[i] = dto.QueuedJob{URL: u}
	}
	if err := q.EnqueueJobs(context.Background(), jobs); err != nil {
		t.Fatalf("EnqueueJobs: %v", err)
	}
}

func TestRun_HappyPath(t *testing.T) {
	q := queue.NewMockQueue()
	enqueueURLs(t, q, []string{"https://example.com/job/1", "https://example.com/job/2"})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var handled []string
	handler := func(ctx context.Context, job dto.QueuedJob) error {
		handled = append(handled, job.URL)
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
	handler := func(ctx context.Context, job dto.QueuedJob) error {
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
	enqueueURLs(t, q, []string{"https://example.com/job/1"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel

	w := instantWorker()
	err := w.run(ctx, q, func(ctx context.Context, job dto.QueuedJob) error { return nil })
	if err != nil {
		t.Errorf("run returned %v, want nil", err)
	}
}

func TestRun_HandlerErrorContinues(t *testing.T) {
	q := queue.NewMockQueue()
	enqueueURLs(t, q, []string{
		"https://example.com/job/1",
		"https://example.com/job/2",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var handled []string
	handler := func(ctx context.Context, job dto.QueuedJob) error {
		handled = append(handled, job.URL)
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

func TestRun_NackOnHandlerError(t *testing.T) {
	q := queue.NewMockQueue()
	enqueueURLs(t, q, []string{"https://example.com/job/1"})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	calls := 0
	handler := func(ctx context.Context, job dto.QueuedJob) error {
		calls++
		cancel()
		return errors.New("handler error")
	}

	w := instantWorker()
	_ = w.run(ctx, q, handler)

	if q.Attempts("https://example.com/job/1") != 1 {
		t.Errorf("want 1 attempt recorded, got %d", q.Attempts("https://example.com/job/1"))
	}
}

func TestRun_ClearAttemptsOnSuccess(t *testing.T) {
	q := queue.NewMockQueue()
	url := "https://example.com/job/1"
	enqueueURLs(t, q, []string{url})

	// Seed an existing attempt count so ClearAttempts has something to clear.
	if err := q.Nack(context.Background(), queue.Item{ID: url}, "failure"); err != nil {
		t.Fatalf("Nack: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := func(ctx context.Context, job dto.QueuedJob) error {
		cancel()
		return nil
	}

	w := instantWorker()
	_ = w.run(ctx, q, handler)

	if got := q.Attempts(url); got != 0 {
		t.Errorf("want 0 attempts after success, got %d", got)
	}
}

func TestRun_DequeueError(t *testing.T) {
	q := queue.NewMockQueue()
	q.DequeueErr = errors.New("valkey unavailable")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel so the errDelay sleep exits immediately

	w := instantWorker()
	err := w.run(ctx, q, func(ctx context.Context, job dto.QueuedJob) error { return nil })
	if err != nil {
		t.Errorf("run returned %v, want nil", err)
	}
}

func TestRunScrapeRequests_NacksFailure(t *testing.T) {
	q := queue.NewMockQueue()
	req := dto.ScrapeRequest{Target: dto.SourceTarget{ID: "request-1", Source: "test"}}
	if err := q.EnqueueScrapeRequest(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	RunScrapeRequests(ctx, q, func(context.Context, dto.ScrapeRequest) error {
		cancel()
		return errors.New("scrape failed")
	})
	if q.Attempts("request-1") != 1 {
		t.Fatalf("attempts = %d, want 1", q.Attempts("request-1"))
	}
}

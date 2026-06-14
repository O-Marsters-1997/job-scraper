package queue

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	tcvalkey "github.com/testcontainers/testcontainers-go/modules/valkey"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

var testAddr string

func TestMain(m *testing.M) {
	ctx := context.Background()

	valkeyContainer, err := tcvalkey.Run(ctx, "valkey/valkey:8")
	if err != nil {
		log.Fatalf("start valkey container: %v", err)
	}

	host, err := valkeyContainer.Host(ctx)
	if err != nil {
		log.Fatalf("container host: %v", err)
	}
	port, err := valkeyContainer.MappedPort(ctx, "6379")
	if err != nil {
		log.Fatalf("container port: %v", err)
	}
	testAddr = fmt.Sprintf("%s:%s", host, port.Port())

	code := m.Run()

	if err := valkeyContainer.Terminate(ctx); err != nil {
		log.Printf("terminate container: %v", err)
	}
	os.Exit(code)
}

func newTestQueue(t *testing.T) *Queue {
	t.Helper()
	q, err := New(testAddr)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		_ = q.client.Do(context.Background(), q.client.B().Flushdb().Build()).Error()
		q.Close()
	})
	return q
}

func jobsFromURLs(urls []string) []dto.QueuedJob {
	jobs := make([]dto.QueuedJob, len(urls))
	for i, u := range urls {
		jobs[i] = dto.QueuedJob{URL: u}
	}
	return jobs
}

func TestEnqueueJobs(t *testing.T) {
	t.Run("deduplicates repeated URLs", func(t *testing.T) {
		ctx := context.Background()
		q := newTestQueue(t)

		url := "https://example.com/job/1"
		if err := q.EnqueueJobs(ctx, jobsFromURLs([]string{url})); err != nil {
			t.Fatalf("first EnqueueJobs: %v", err)
		}
		if err := q.EnqueueJobs(ctx, jobsFromURLs([]string{url})); err != nil {
			t.Fatalf("second EnqueueJobs: %v", err)
		}

		got, err := q.client.Do(ctx, q.client.B().Zcard().Key(sortedSetKey).Build()).AsInt64()
		if err != nil {
			t.Fatalf("ZCARD: %v", err)
		}
		if got != 1 {
			t.Errorf("want 1 member in set after duplicate enqueue, got %d", got)
		}
	})

	t.Run("batch", func(t *testing.T) {
		ctx := context.Background()
		q := newTestQueue(t)

		urls := make([]string, 20)
		for i := range urls {
			urls[i] = fmt.Sprintf("https://example.com/job/%d", i)
		}

		if err := q.EnqueueJobs(ctx, jobsFromURLs(urls)); err != nil {
			t.Fatalf("EnqueueJobs: %v", err)
		}

		got, err := q.client.Do(ctx, q.client.B().Zcard().Key(sortedSetKey).Build()).AsInt64()
		if err != nil {
			t.Fatalf("ZCARD: %v", err)
		}
		if int(got) != len(urls) {
			t.Errorf("want %d members, got %d", len(urls), got)
		}
	})

	t.Run("nil and empty slice", func(t *testing.T) {
		ctx := context.Background()
		q := newTestQueue(t)

		if err := q.EnqueueJobs(ctx, nil); err != nil {
			t.Errorf("EnqueueJobs(nil): %v", err)
		}
		if err := q.EnqueueJobs(ctx, []dto.QueuedJob{}); err != nil {
			t.Errorf("EnqueueJobs([]): %v", err)
		}
	})

	t.Run("payload round-trips through hash", func(t *testing.T) {
		ctx := context.Background()
		q := newTestQueue(t)

		job := dto.QueuedJob{URL: "https://example.com/job/payload", Relevance: 75}
		if err := q.EnqueueJobs(ctx, []dto.QueuedJob{job}); err != nil {
			t.Fatalf("EnqueueJobs: %v", err)
		}

		got, ok, err := q.Dequeue(ctx)
		if err != nil {
			t.Fatalf("Dequeue: %v", err)
		}
		if !ok {
			t.Fatal("want ok=true")
		}
		if got.URL != job.URL {
			t.Errorf("URL: got %q, want %q", got.URL, job.URL)
		}
		if got.Relevance != job.Relevance {
			t.Errorf("Relevance: got %d, want %d", got.Relevance, job.Relevance)
		}
	})
}

func TestDequeue(t *testing.T) {
	t.Run("empty queue", func(t *testing.T) {
		ctx := context.Background()
		q := newTestQueue(t)

		job, ok, err := q.Dequeue(ctx)
		if err != nil {
			t.Fatalf("Dequeue: %v", err)
		}
		if ok || job.URL != "" {
			t.Errorf("want (zero, false, nil) on empty queue, got (%v %v %v)", job, ok, err)
		}
	})

	t.Run("atomic — only one concurrent caller wins", func(t *testing.T) {
		ctx := context.Background()
		q := newTestQueue(t)

		url := "https://example.com/job/atomic"
		if err := q.EnqueueJobs(ctx, jobsFromURLs([]string{url})); err != nil {
			t.Fatalf("EnqueueJobs: %v", err)
		}

		const workers = 10
		results := make([]string, 0, workers)
		var mu sync.Mutex
		var wg sync.WaitGroup

		for range workers {
			wg.Go(func() {
				job, ok, err := q.Dequeue(ctx)
				if err != nil {
					t.Errorf("Dequeue error: %v", err)
					return
				}
				if ok {
					mu.Lock()
					results = append(results, job.URL)
					mu.Unlock()
				}
			})
		}
		wg.Wait()

		if len(results) != 1 {
			t.Errorf("want exactly 1 dequeue result, got %d: %v", len(results), results)
		}
	})
}

func TestNack(t *testing.T) {
	t.Run("increments attempt counter and re-queues with backoff", func(t *testing.T) {
		ctx := context.Background()
		q := newTestQueue(t)
		url := "https://example.com/job/nack1"

		if err := q.Nack(ctx, url); err != nil {
			t.Fatalf("Nack: %v", err)
		}

		count, err := q.client.Do(ctx, q.client.B().Hget().Key(attemptsKey).Field(url).Build()).AsInt64()
		if err != nil {
			t.Fatalf("HGET attempts: %v", err)
		}
		if count != 1 {
			t.Errorf("want attempt count 1, got %d", count)
		}

		card, err := q.client.Do(ctx, q.client.B().Zcard().Key(sortedSetKey).Build()).AsInt64()
		if err != nil {
			t.Fatalf("ZCARD: %v", err)
		}
		if card != 1 {
			t.Errorf("want 1 member re-queued in pending set, got %d", card)
		}
	})

	t.Run("dead-letters after maxAttempts and clears counter", func(t *testing.T) {
		ctx := context.Background()
		q := newTestQueue(t)
		url := "https://example.com/job/nack2"

		for range maxAttempts {
			if err := q.Nack(ctx, url); err != nil {
				t.Fatalf("Nack: %v", err)
			}
		}

		dlCard, err := q.client.Do(ctx, q.client.B().Zcard().Key(deadLetterKey).Build()).AsInt64()
		if err != nil {
			t.Fatalf("ZCARD deadletter: %v", err)
		}
		if dlCard != 1 {
			t.Errorf("want 1 member in dead-letter set, got %d", dlCard)
		}

		exists, err := q.client.Do(ctx, q.client.B().Hexists().Key(attemptsKey).Field(url).Build()).AsBool()
		if err != nil {
			t.Fatalf("HEXISTS attempts: %v", err)
		}
		if exists {
			t.Error("want attempt counter cleared after dead-lettering")
		}
	})

	t.Run("no duplicate entries — re-queuing does not stack", func(t *testing.T) {
		ctx := context.Background()
		q := newTestQueue(t)
		url := "https://example.com/job/nack3"

		// Nack twice but below maxAttempts; URL should appear once in pending.
		for range 2 {
			if err := q.Nack(ctx, url); err != nil {
				t.Fatalf("Nack: %v", err)
			}
		}

		card, err := q.client.Do(ctx, q.client.B().Zcard().Key(sortedSetKey).Build()).AsInt64()
		if err != nil {
			t.Fatalf("ZCARD: %v", err)
		}
		if card != 1 {
			t.Errorf("want 1 member in pending (ZADD overwrites score), got %d", card)
		}
	})
}

func TestClearAttempts(t *testing.T) {
	t.Run("removes attempt counter for url", func(t *testing.T) {
		ctx := context.Background()
		q := newTestQueue(t)
		url := "https://example.com/job/clear1"

		if err := q.Nack(ctx, url); err != nil {
			t.Fatalf("Nack: %v", err)
		}

		if err := q.ClearAttempts(ctx, url); err != nil {
			t.Fatalf("ClearAttempts: %v", err)
		}

		exists, err := q.client.Do(ctx, q.client.B().Hexists().Key(attemptsKey).Field(url).Build()).AsBool()
		if err != nil {
			t.Fatalf("HEXISTS: %v", err)
		}
		if exists {
			t.Error("want attempt counter removed after ClearAttempts")
		}
	})
}

func TestGetLastScraped(t *testing.T) {
	t.Run("returns time within window after set", func(t *testing.T) {
		ctx := context.Background()
		q := newTestQueue(t)

		source := "test-source"

		before := time.Now().Truncate(time.Millisecond)
		if err := q.SetLastScraped(ctx, source); err != nil {
			t.Fatalf("SetLastScraped: %v", err)
		}
		after := time.Now().Truncate(time.Millisecond).Add(time.Millisecond)

		got, ok, err := q.GetLastScraped(ctx, source)
		if err != nil {
			t.Fatalf("GetLastScraped: %v", err)
		}
		if !ok {
			t.Fatal("want ok=true, got false")
		}
		if got.Before(before) || got.After(after) {
			t.Errorf("got time %v outside expected window [%v, %v]", got, before, after)
		}
	})

	t.Run("returns zero and false when never scraped", func(t *testing.T) {
		ctx := context.Background()
		q := newTestQueue(t)

		got, ok, err := q.GetLastScraped(ctx, "unknown-source")
		if err != nil {
			t.Fatalf("GetLastScraped: %v", err)
		}
		if ok || !got.IsZero() {
			t.Errorf("want (zero, false, nil) for unknown source, got (%v %v %v)", got, ok, err)
		}
	})
}

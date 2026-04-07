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

// newTestQueue returns a Queue connected to the test Valkey instance.
// It registers a Cleanup to flush the DB and close the connection after the test.
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

func TestEnqueue(t *testing.T) {
	t.Run("deduplicates repeated URLs", func(t *testing.T) {
		ctx := context.Background()
		q := newTestQueue(t)

		url := "https://example.com/job/1"
		if err := q.Enqueue(ctx, []string{url}); err != nil {
			t.Fatalf("first Enqueue: %v", err)
		}
		if err := q.Enqueue(ctx, []string{url}); err != nil {
			t.Fatalf("second Enqueue: %v", err)
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

		if err := q.Enqueue(ctx, urls); err != nil {
			t.Fatalf("Enqueue: %v", err)
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

		if err := q.Enqueue(ctx, nil); err != nil {
			t.Errorf("Enqueue(nil): %v", err)
		}
		if err := q.Enqueue(ctx, []string{}); err != nil {
			t.Errorf("Enqueue([]): %v", err)
		}
	})
}

func TestDequeue(t *testing.T) {
	t.Run("empty queue", func(t *testing.T) {
		ctx := context.Background()
		q := newTestQueue(t)

		url, ok, err := q.Dequeue(ctx)
		if err != nil {
			t.Fatalf("Dequeue: %v", err)
		}
		if ok || url != "" {
			t.Errorf("want (\"\" false nil) on empty queue, got (%q %v %v)", url, ok, err)
		}
	})

	t.Run("atomic — only one concurrent caller wins", func(t *testing.T) {
		ctx := context.Background()
		q := newTestQueue(t)

		url := "https://example.com/job/atomic"
		if err := q.Enqueue(ctx, []string{url}); err != nil {
			t.Fatalf("Enqueue: %v", err)
		}

		const workers = 10
		results := make([]string, 0, workers)
		var mu sync.Mutex
		var wg sync.WaitGroup

		for range workers {
			wg.Go(func() {
				u, ok, err := q.Dequeue(ctx)
				if err != nil {
					t.Errorf("Dequeue error: %v", err)
					return
				}
				if ok {
					mu.Lock()
					results = append(results, u)
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

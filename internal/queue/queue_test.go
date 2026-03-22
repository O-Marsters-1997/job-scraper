package queue

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testAddr string

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "valkey/valkey:8",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForLog("Ready to accept connections"),
		},
		Started: true,
	})
	if err != nil {
		log.Fatalf("start valkey container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		log.Fatalf("container host: %v", err)
	}
	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		log.Fatalf("container port: %v", err)
	}
	testAddr = fmt.Sprintf("%s:%s", host, port.Port())

	code := m.Run()

	if err := container.Terminate(ctx); err != nil {
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

func TestEnqueue_Deduplication(t *testing.T) {
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
}

func TestEnqueue_Batch(t *testing.T) {
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
}

func TestEnqueue_Empty(t *testing.T) {
	ctx := context.Background()
	q := newTestQueue(t)

	if err := q.Enqueue(ctx, nil); err != nil {
		t.Errorf("Enqueue(nil): %v", err)
	}
	if err := q.Enqueue(ctx, []string{}); err != nil {
		t.Errorf("Enqueue([]): %v", err)
	}
}

func TestDequeue_Empty(t *testing.T) {
	ctx := context.Background()
	q := newTestQueue(t)

	url, ok, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue: %v", err)
	}
	if ok || url != "" {
		t.Errorf("want (\"\" false nil) on empty queue, got (%q %v %v)", url, ok, err)
	}
}

func TestDequeue_Atomic(t *testing.T) {
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
}

func TestSetLastScraped_GetLastScraped(t *testing.T) {
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
}

func TestGetLastScraped_NeverScraped(t *testing.T) {
	ctx := context.Background()
	q := newTestQueue(t)

	got, ok, err := q.GetLastScraped(ctx, "unknown-source")
	if err != nil {
		t.Fatalf("GetLastScraped: %v", err)
	}
	if ok || !got.IsZero() {
		t.Errorf("want (zero, false, nil) for unknown source, got (%v %v %v)", got, ok, err)
	}
}

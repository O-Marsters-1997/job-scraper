package queue

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	tcvalkey "github.com/testcontainers/testcontainers-go/modules/valkey"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

var testAddr string

func TestMain(m *testing.M) {
	ctx := context.Background()
	container, err := tcvalkey.Run(ctx, "valkey/valkey:8")
	if err != nil {
		log.Fatalf("start valkey container: %v", err)
	}
	host, err := container.Host(ctx)
	if err != nil {
		log.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		log.Fatal(err)
	}
	testAddr = fmt.Sprintf("%s:%s", host, port.Port())
	code := m.Run()
	if err := container.Terminate(ctx); err != nil {
		log.Printf("terminate container: %v", err)
	}
	os.Exit(code)
}

func newTestQueue(t *testing.T) *Queue {
	t.Helper()
	q, err := New(testAddr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = q.client.Do(context.Background(), q.client.B().Flushdb().Build()).Error()
		q.Close()
	})
	return q
}

func TestEnqueueJobs(t *testing.T) {
	ctx := context.Background()
	q := newTestQueue(t)
	job := dto.QueuedJob{URL: "https://example.com/job/1", Relevance: 75}
	if err := q.EnqueueJobs(ctx, []dto.QueuedJob{job, job}); err != nil {
		t.Fatal(err)
	}
	item, ok, err := q.ClaimReady(ctx, Detail, time.Minute)
	if err != nil || !ok || item.ID != job.URL {
		t.Fatalf("claim: %+v %v %v", item, ok, err)
	}
	if _, ok, err := q.ClaimReady(ctx, Detail, time.Minute); err != nil || ok {
		t.Fatalf("duplicate claim: %v %v", ok, err)
	}
}

func TestLastScraped(t *testing.T) {
	ctx := context.Background()
	q := newTestQueue(t)
	if _, ok, err := q.GetLastScraped(ctx, "source"); err != nil || ok {
		t.Fatalf("initial: %v %v", ok, err)
	}
	if err := q.SetLastScraped(ctx, "source"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := q.GetLastScraped(ctx, "source"); err != nil || !ok {
		t.Fatalf("set: %v %v", ok, err)
	}
}

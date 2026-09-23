package queue

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcvalkey "github.com/testcontainers/testcontainers-go/modules/valkey"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestRecoverableClaim(t *testing.T) {
	ctx := context.Background()
	q := newTestQueue(t)
	now := time.Now()
	if err := q.Publish(ctx, Detail, "https://example.com/job-1", []byte(`{"url":"job-1","card":"kept"}`), now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := q.ClaimReady(ctx, Detail, time.Second); err != nil || ok {
		t.Fatalf("future work claimed: ok=%v err=%v", ok, err)
	}
	if err := q.Publish(ctx, Detail, "https://example.com/job-1", []byte(`{"url":"wrong"}`), now); err != nil {
		t.Fatal(err)
	}
	if err := q.Publish(ctx, Detail, "https://example.com/job-2", []byte(`{"url":"job-2","card":"kept"}`), now); err != nil {
		t.Fatal(err)
	}
	stats, err := q.Stats(ctx, Detail)
	if err != nil || stats.Ready != 2 || stats.OldestDueAt.IsZero() {
		t.Fatalf("stats = %+v, %v", stats, err)
	}
	first, ok, err := q.ClaimReady(ctx, Detail, time.Millisecond)
	if err != nil || !ok || first.ID != "https://example.com/job-2" {
		t.Fatalf("claim = %+v, %v, %v", first, ok, err)
	}
	q.client.Do(ctx, q.client.B().Zadd().Key("queue:detail:leased").ScoreMember().ScoreMember(float64(time.Now().Add(-time.Second).UnixMilli()), first.ID).Build())
	second, ok, err := q.ClaimReady(ctx, Detail, time.Minute)
	if err != nil || !ok || second.ID != first.ID || string(second.Payload) != string(first.Payload) || second.Token == first.Token {
		t.Fatalf("reclaim = %+v, %v, %v", second, ok, err)
	}
	if err := q.Ack(ctx, first); err == nil {
		t.Fatal("stale claim acknowledged current work")
	}
	if err := q.Ack(ctx, second); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := q.ClaimReady(ctx, Detail, time.Minute); err != nil || ok {
		t.Fatalf("acknowledged work claimed: ok=%v err=%v", ok, err)
	}
	if err := q.Publish(ctx, Detail, second.ID, []byte("new rerun"), time.Now()); err != nil {
		t.Fatal(err)
	}
	rerun, ok, err := q.ClaimReady(ctx, Detail, time.Minute)
	if err != nil || !ok || string(rerun.Payload) != "new rerun" {
		t.Fatalf("explicit rerun = %+v, %v, %v", rerun, ok, err)
	}
}

func TestDeadLetterReplay(t *testing.T) {
	ctx := context.Background()
	q := newTestQueue(t)
	payload := []byte(`{"target":"original"}`)
	if err := q.Publish(ctx, ScrapeRequest, "request-1", payload, time.Now()); err != nil {
		t.Fatal(err)
	}
	for range maxAttempts {
		item, ok, err := q.ClaimReady(ctx, ScrapeRequest, time.Minute)
		if err != nil || !ok {
			t.Fatalf("claim: %v %v", ok, err)
		}
		if err := q.Nack(ctx, item, "temporary"); err != nil {
			t.Fatal(err)
		}
		if item.Attempts < maxAttempts-1 {
			q.client.Do(ctx, q.client.B().Zadd().Key(readyKey(ScrapeRequest)).ScoreMember().ScoreMember(float64(time.Now().Add(-time.Second).UnixMilli()), item.ID).Build())
		}
	}
	dead, err := q.DeadLetters(ctx, ScrapeRequest)
	if err != nil || len(dead) != 1 || string(dead[0].Payload) != string(payload) || dead[0].Failure != "temporary" {
		t.Fatalf("dead letters = %+v, %v", dead, err)
	}
	stats, err := q.Stats(ctx, ScrapeRequest)
	if err != nil || stats.Dead != 1 || stats.Ready != 0 {
		t.Fatalf("dead stats = %+v, %v", stats, err)
	}
	if err := q.ReplayDeadLetter(ctx, ScrapeRequest, "request-1"); err != nil {
		t.Fatal(err)
	}
	item, ok, err := q.ClaimReady(ctx, ScrapeRequest, time.Minute)
	if err != nil || !ok || string(item.Payload) != string(payload) {
		t.Fatalf("replayed claim = %+v, %v, %v", item, ok, err)
	}
}

func TestConcurrentClaimHasOneWinner(t *testing.T) {
	ctx := context.Background()
	q := newTestQueue(t)
	if err := q.Publish(ctx, Detail, "https://example.com/one", []byte("one"), time.Now()); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wins := make(chan Item, 10)
	for range 10 {
		wg.Go(func() {
			item, ok, err := q.ClaimReady(ctx, Detail, time.Minute)
			if err != nil {
				t.Error(err)
			} else if ok {
				wins <- item
			}
		})
	}
	wg.Wait()
	if len(wins) != 1 {
		t.Fatalf("claimed %d times; want one", len(wins))
	}
}

func TestDetailPublishNormalizesURLIdentity(t *testing.T) {
	ctx := context.Background()
	q := newTestQueue(t)
	if err := q.Publish(ctx, Detail, "HTTPS://EXAMPLE.COM/path#first", []byte("first"), time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := q.Publish(ctx, Detail, "https://example.com/path#second", []byte("second"), time.Now()); err != nil {
		t.Fatal(err)
	}
	item, ok, err := q.ClaimReady(ctx, Detail, time.Minute)
	if err != nil || !ok || string(item.Payload) != "first" {
		t.Fatalf("claim: %+v %v %v", item, ok, err)
	}
	if _, ok, err := q.ClaimReady(ctx, Detail, time.Minute); err != nil || ok {
		t.Fatalf("duplicate: %v %v", ok, err)
	}
}

func TestScrapeRequestDeduplicatesUntilAck(t *testing.T) {
	ctx := context.Background()
	q := newTestQueue(t)
	req := dto.ScrapeRequest{Target: dto.SourceTarget{ID: "target-1"}}
	if err := q.EnqueueScrapeRequest(ctx, req); err != nil {
		t.Fatal(err)
	}
	if err := q.EnqueueScrapeRequest(ctx, req); err != nil {
		t.Fatal(err)
	}
	item, ok, err := q.ClaimReady(ctx, ScrapeRequest, time.Minute)
	if err != nil || !ok {
		t.Fatalf("first claim: %v %v", ok, err)
	}
	if _, ok, err := q.ClaimReady(ctx, ScrapeRequest, time.Minute); err != nil || ok {
		t.Fatalf("duplicate claim: %v %v", ok, err)
	}
	if err := q.Ack(ctx, item); err != nil {
		t.Fatal(err)
	}
	if err := q.EnqueueScrapeRequest(ctx, req); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := q.ClaimReady(ctx, ScrapeRequest, time.Minute); err != nil || !ok {
		t.Fatalf("later rerun: %v %v", ok, err)
	}
}

func TestExplicitScrapeRerunReplaysDeadRequest(t *testing.T) {
	ctx := context.Background()
	q := newTestQueue(t)
	req := dto.ScrapeRequest{Target: dto.SourceTarget{ID: "target-1"}}
	if err := q.EnqueueScrapeRequest(ctx, req); err != nil {
		t.Fatal(err)
	}
	for attempt := range maxAttempts {
		item, ok, err := q.ClaimReady(ctx, ScrapeRequest, time.Minute)
		if err != nil || !ok {
			t.Fatalf("claim %d: %v %v", attempt, ok, err)
		}
		if err := q.Nack(ctx, item, "failed"); err != nil {
			t.Fatal(err)
		}
		if attempt < maxAttempts-1 {
			q.client.Do(ctx, q.client.B().Zadd().Key(readyKey(ScrapeRequest)).ScoreMember().ScoreMember(float64(time.Now().Add(-time.Second).UnixMilli()), item.ID).Build())
		}
	}
	if err := q.EnqueueScrapeRequest(ctx, req); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := q.ClaimReady(ctx, ScrapeRequest, time.Minute); err != nil || !ok {
		t.Fatalf("explicit rerun after dead letter: %v %v", ok, err)
	}
}

func TestAOFRestartRetainsPendingAndAck(t *testing.T) {
	ctx := context.Background()
	container, err := tcvalkey.Run(ctx, "valkey/valkey:8", testcontainers.WithCmdArgs("--appendonly", "yes", "--appendfsync", "always"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatal(err)
	}
	addr := fmt.Sprintf("%s:%s", host, port.Port())
	q, err := New(addr)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"https://example.com/ack", "https://example.com/pending"} {
		if err := q.Publish(ctx, Detail, id, []byte(id), time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	item, ok, err := q.ClaimReady(ctx, Detail, time.Minute)
	if err != nil || !ok || item.ID != "https://example.com/ack" {
		t.Fatalf("claim: %+v %v %v", item, ok, err)
	}
	if err := q.Ack(ctx, item); err != nil {
		t.Fatal(err)
	}
	q.Close()
	if err := container.Stop(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if err := container.Start(ctx); err != nil {
		t.Fatal(err)
	}
	port, err = container.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatal(err)
	}
	addr = fmt.Sprintf("%s:%s", host, port.Port())
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		q, err = New(addr)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()
	item, ok, err = q.ClaimReady(ctx, Detail, time.Minute)
	if err != nil || !ok || item.ID != "https://example.com/pending" || string(item.Payload) != "https://example.com/pending" {
		t.Fatalf("surviving claim: %+v %v %v", item, ok, err)
	}
	if _, ok, err := q.ClaimReady(ctx, Detail, time.Minute); err != nil || ok {
		t.Fatalf("acknowledged item reappeared: %v %v", ok, err)
	}
}

package queue

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestSourceTaggedDetailUsesLane(t *testing.T) {
	ctx := context.Background()
	q := newTestQueue(t)
	job := dto.QueuedJob{URL: "https://example.com/job", Card: dto.Job{Source: "wis"}}
	if err := q.EnqueueJobs(ctx, []dto.QueuedJob{job}); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := q.ClaimReady(ctx, Detail, time.Minute); err != nil || ok {
		t.Fatalf("legacy claim = %v, %v", ok, err)
	}
	item, ok, err := q.ClaimSource(ctx, time.Minute)
	if err != nil || !ok || item.Source != "wis" || item.Kind != SourceDetail {
		t.Fatalf("source claim = %+v, %v, %v", item, ok, err)
	}
	var got dto.QueuedJob
	if err := json.Unmarshal(item.Payload, &got); err != nil || got.URL != job.URL {
		t.Fatalf("payload = %+v, %v", got, err)
	}
}

func TestSourceClaimPriorityAndDueTime(t *testing.T) {
	ctx := context.Background()
	q := newTestQueue(t)
	now := time.Now()
	for _, task := range []struct {
		kind SourceKind
		id   string
		due  time.Time
	}{
		{SourceDetail, "old-detail", now.Add(-time.Minute)},
		{SourceListing, "listing", now},
		{SourceListing, "delayed-listing", now.Add(time.Hour)},
	} {
		if err := q.PublishSource(ctx, "wis", task.kind, task.id, []byte(task.id), task.due, 0); err != nil {
			t.Fatal(err)
		}
	}
	first, ok, err := q.ClaimSource(ctx, time.Minute)
	if err != nil || !ok || first.ID != "listing" {
		t.Fatalf("first = %+v, %v, %v", first, ok, err)
	}
	if err := q.AckSource(ctx, first); err != nil {
		t.Fatal(err)
	}
	second, ok, err := q.ClaimSource(ctx, time.Minute)
	if err != nil || !ok || second.ID != "old-detail" {
		t.Fatalf("second = %+v, %v, %v", second, ok, err)
	}
}

func TestSourceClaimLeaseAndCooldown(t *testing.T) {
	ctx := context.Background()
	q := newTestQueue(t)
	other, err := New(testAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	for _, task := range []struct{ source, id string }{{"wis", "one"}, {"wis", "two"}, {"remoteok", "three"}} {
		if err := q.PublishSource(ctx, task.source, SourceDetail, task.id, []byte(task.id), time.Now(), time.Hour); err != nil {
			t.Fatal(err)
		}
	}
	first, ok, err := q.ClaimSource(ctx, time.Minute)
	if err != nil || !ok {
		t.Fatalf("first claim: %v %v", ok, err)
	}
	second, ok, err := other.ClaimSource(ctx, time.Minute)
	if err != nil || !ok || second.Source == first.Source {
		t.Fatalf("second claim = %+v, %v, %v", second, ok, err)
	}
	if _, ok, err := q.ClaimSource(ctx, time.Minute); err != nil || ok {
		t.Fatalf("overlapping claim: %v %v", ok, err)
	}
	if err := q.AckSource(ctx, first); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := q.ClaimSource(ctx, time.Minute); err != nil || ok {
		t.Fatalf("cooldown claim: %v %v", ok, err)
	}
}

func TestSourceClaimExpiresAndFencesOldOwner(t *testing.T) {
	ctx := context.Background()
	q := newTestQueue(t)
	if err := q.PublishSource(ctx, "wis", SourceDetail, "job", []byte("job"), time.Now(), 0); err != nil {
		t.Fatal(err)
	}
	first, ok, err := q.ClaimSource(ctx, time.Minute)
	if err != nil || !ok {
		t.Fatalf("first claim: %v %v", ok, err)
	}
	if err := q.RenewSource(ctx, first, time.Minute); err != nil {
		t.Fatal(err)
	}
	q.client.Do(ctx, q.client.B().Zadd().Key("queue:source:leases").ScoreMember().ScoreMember(float64(time.Now().Add(-time.Second).UnixMilli()), first.Source).Build())
	q.client.Do(ctx, q.client.B().Zadd().Key("queue:source:sources").ScoreMember().ScoreMember(float64(time.Now().Add(-time.Second).UnixMilli()), first.Source).Build())
	if err := q.RenewSource(ctx, first, time.Minute); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("expired owner renewed lease: %v", err)
	}
	if _, ok, err := q.ClaimSource(ctx, time.Minute); err != nil || ok {
		t.Fatalf("expired claim should retry later: %v %v", ok, err)
	}
	q.client.Do(ctx, q.client.B().Zadd().Key("queue:source:ready:wis:detail").ScoreMember().ScoreMember(float64(time.Now().Add(-time.Second).UnixMilli()), first.ID).Build())
	q.client.Do(ctx, q.client.B().Zadd().Key("queue:source:sources").ScoreMember().ScoreMember(float64(time.Now().Add(-time.Second).UnixMilli()), first.Source).Build())
	second, ok, err := q.ClaimSource(ctx, time.Minute)
	if err != nil || !ok || second.Token == first.Token {
		t.Fatalf("reclaim = %+v, %v, %v", second, ok, err)
	}
	if err := q.AckSource(ctx, first); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("stale owner ack = %v", err)
	}
	if err := q.AckSource(ctx, second); err != nil {
		t.Fatal(err)
	}
}

func TestPausedSourceDoesNotBlockDirectSource(t *testing.T) {
	ctx := context.Background()
	q := newTestQueue(t)
	for _, source := range []string{"indeed", "remoteok"} {
		if err := q.PublishSource(ctx, source, SourceListing, source, []byte(source), time.Now(), 0); err != nil {
			t.Fatal(err)
		}
	}
	if err := q.PauseSource(ctx, "indeed", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	item, ok, err := q.ClaimSource(ctx, time.Minute)
	if err != nil || !ok || item.Source != "remoteok" {
		t.Fatalf("direct source claim = %+v, %v, %v", item, ok, err)
	}
	if err := q.PauseSource(ctx, "indeed", time.Now()); err != nil {
		t.Fatal(err)
	}
	item, ok, err = q.ClaimSource(ctx, time.Minute)
	if err != nil || !ok || item.Source != "indeed" {
		t.Fatalf("resumed source claim = %+v, %v, %v", item, ok, err)
	}
}

func TestSourceFailuresReachDeadLetterCount(t *testing.T) {
	ctx := context.Background()
	q := newTestQueue(t)
	if err := q.PublishSource(ctx, "wis", SourceDetail, "failed-job", []byte("job"), time.Now(), 0); err != nil {
		t.Fatal(err)
	}
	for attempt := range maxAttempts {
		item, ok, err := q.ClaimSource(ctx, time.Minute)
		if err != nil || !ok {
			t.Fatalf("claim: %v %v", ok, err)
		}
		if err := q.NackSource(ctx, item, "fetch failed"); err != nil {
			t.Fatal(err)
		}
		if attempt < maxAttempts-1 {
			q.client.Do(ctx, q.client.B().Zadd().Key("queue:source:ready:wis:detail").ScoreMember().ScoreMember(float64(time.Now().Add(-time.Second).UnixMilli()), item.ID).Build())
			q.client.Do(ctx, q.client.B().Zadd().Key("queue:source:sources").ScoreMember().ScoreMember(float64(time.Now().Add(-time.Second).UnixMilli()), item.Source).Build())
		}
	}
	count, err := q.DeadLetterCount(ctx)
	if err != nil || count != 1 {
		t.Fatalf("dead count = %d, %v", count, err)
	}
}

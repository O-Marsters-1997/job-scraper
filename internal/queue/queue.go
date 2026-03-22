package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/valkey-io/valkey-go"
)

const (
	sortedSetKey      = "jobs:pending"
	lastScrapedKeyFmt = "scrape:last:%s"
)

// JobQueue is the boundary all callers depend on.
// Timing decisions (time.Now, UnixMilli conversion) are owned by the implementation.
type JobQueue interface {
	// Enqueue adds urls for immediate processing.
	// ZADD NX semantics: already-queued URLs are silently skipped.
	// All urls are enqueued in a single command.
	Enqueue(ctx context.Context, urls []string) error

	// Dequeue atomically removes and returns the next ready URL.
	// Uses ZPOPMIN — guaranteed safe for concurrent workers.
	// Returns ("", false, nil) when nothing is ready.
	Dequeue(ctx context.Context) (string, bool, error)

	SetLastScraped(ctx context.Context, source string) error

	// GetLastScraped returns the time of the last successful scrape for source.
	// Returns (zero, false, nil) when the source has never been scraped.
	GetLastScraped(ctx context.Context, source string) (time.Time, bool, error)

	Close()
}

// Queue wraps a Valkey client and owns the jobs pending sorted set.
type Queue struct {
	client valkey.Client
}

// New connects to Valkey at addr and verifies connectivity with PING.
func New(addr string) (*Queue, error) {
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{addr},
	})
	if err != nil {
		return nil, fmt.Errorf("valkey connect: %w", err)
	}

	if err := client.Do(context.Background(), client.B().Ping().Build()).Error(); err != nil {
		client.Close()
		return nil, fmt.Errorf("valkey ping: %w", err)
	}

	return &Queue{client: client}, nil
}

func (q *Queue) Enqueue(ctx context.Context, urls []string) error {
	if len(urls) == 0 {
		return nil
	}

	score := float64(time.Now().UnixMilli())
	sm := q.client.B().Zadd().Key(sortedSetKey).Nx().ScoreMember()
	for _, url := range urls {
		sm = sm.ScoreMember(score, url)
	}

	return q.client.Do(ctx, sm.Build()).Error()
}

func (q *Queue) Dequeue(ctx context.Context) (string, bool, error) {
	cmd := q.client.B().Zpopmin().Key(sortedSetKey).Count(1).Build()
	scores, err := q.client.Do(ctx, cmd).AsZScores()
	if err != nil {
		return "", false, fmt.Errorf("zpopmin: %w", err)
	}
	if len(scores) == 0 {
		return "", false, nil
	}

	return scores[0].Member, true, nil
}

func (q *Queue) SetLastScraped(ctx context.Context, source string) error {
	key := fmt.Sprintf(lastScrapedKeyFmt, source)
	val := fmt.Sprintf("%d", time.Now().UnixMilli())
	cmd := q.client.B().Set().Key(key).Value(val).Build()
	return q.client.Do(ctx, cmd).Error()
}

func (q *Queue) GetLastScraped(ctx context.Context, source string) (time.Time, bool, error) {
	key := fmt.Sprintf(lastScrapedKeyFmt, source)
	cmd := q.client.B().Get().Key(key).Build()
	val, err := q.client.Do(ctx, cmd).AsInt64()
	if err != nil {
		if valkey.IsValkeyNil(err) {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, fmt.Errorf("get last scraped %s: %w", source, err)
	}
	return time.UnixMilli(val), true, nil
}

func (q *Queue) Close() {
	q.client.Close()
}

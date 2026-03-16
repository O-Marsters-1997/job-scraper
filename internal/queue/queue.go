package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/valkey-io/valkey-go"
)

const (
	sortedSetKey     = "jobs:pending"
	lastScrapedKeyFmt = "scrape:last:%s"
)

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

// Enqueue adds each URL to the sorted set with score = readyAt.UnixMilli().
// ZADD NX ensures existing members are never overwritten — URLs already in
// the set are silently skipped, providing deduplication at the queue level.
func (q *Queue) Enqueue(ctx context.Context, urls []string, readyAt time.Time) error {
	score := float64(readyAt.UnixMilli())

	for _, url := range urls {
		cmd := q.client.B().Zadd().Key(sortedSetKey).Nx().
			ScoreMember().ScoreMember(score, url).
			Build()

		if err := q.client.Do(ctx, cmd).Error(); err != nil {
			return fmt.Errorf("zadd %s: %w", url, err)
		}
	}

	return nil
}

// Dequeue fetches the next ready URL (score <= now) and removes it from the
// sorted set. Returns ("", false, nil) when the queue is empty.
func (q *Queue) Dequeue(ctx context.Context) (string, bool, error) {
	nowMs := float64(time.Now().UnixMilli())

	rangeCmd := q.client.B().Zrangebyscore().Key(sortedSetKey).
		Min("0").Max(fmt.Sprintf("%g", nowMs)).
		Limit(0, 1).
		Build()

	urls, err := q.client.Do(ctx, rangeCmd).AsStrSlice()
	if err != nil {
		return "", false, fmt.Errorf("zrangebyscore: %w", err)
	}
	if len(urls) == 0 {
		return "", false, nil
	}

	url := urls[0]
	remCmd := q.client.B().Zrem().Key(sortedSetKey).Member(url).Build()
	if err := q.client.Do(ctx, remCmd).Error(); err != nil {
		return "", false, fmt.Errorf("zrem: %w", err)
	}

	return url, true, nil
}

// SetLastScraped records the time of the last successful scrape for a source.
func (q *Queue) SetLastScraped(ctx context.Context, source string, t time.Time) error {
	key := fmt.Sprintf(lastScrapedKeyFmt, source)
	val := fmt.Sprintf("%d", t.UnixMilli())
	cmd := q.client.B().Set().Key(key).Value(val).Build()
	return q.client.Do(ctx, cmd).Error()
}

// GetLastScraped returns the time of the last successful scrape for a source.
// ok is false when the key does not exist (source has never been scraped).
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

// Close closes the underlying Valkey connection.
func (q *Queue) Close() {
	q.client.Close()
}

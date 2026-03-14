package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/valkey-io/valkey-go"
)

const sortedSetKey = "jobs:pending"

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

// Close closes the underlying Valkey connection.
func (q *Queue) Close() {
	q.client.Close()
}

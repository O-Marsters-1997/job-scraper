package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	sortedSetKey      = "jobs:pending"
	payloadKey        = "jobs:payload"
	attemptsKey       = "jobs:attempts"
	deadLetterKey     = "jobs:deadletter"
	lastScrapedKeyFmt = "scrape:last:%s"
	backoffBase       = 30 * time.Second
)

// JobQueue is the boundary all callers depend on.
// Timing decisions (time.Now, UnixMilli conversion) are owned by the implementation.
type JobQueue interface {
	// EnqueueJobs stores each job's URL in the sorted set (ZADD NX — already-queued
	// URLs are silently skipped) and persists the full payload in a hash.
	EnqueueJobs(ctx context.Context, jobs []dto.QueuedJob) error

	// Dequeue atomically removes and returns the next ready job.
	// Uses ZPOPMIN — guaranteed safe for concurrent workers.
	// Returns (zero, false, nil) when nothing is ready.
	Dequeue(ctx context.Context) (dto.QueuedJob, bool, error)

	SetLastScraped(ctx context.Context, source string) error

	// Returns (zero, false, nil) when the source has never been scraped.
	GetLastScraped(ctx context.Context, source string) (time.Time, bool, error)

	// Nack increments the attempt counter for url. If the counter reaches
	// maxAttempts the URL is moved to the dead-letter sorted set and its
	// attempt counter is cleared; otherwise it is re-queued with a backoff
	// delay proportional to the attempt count.
	Nack(ctx context.Context, url string, maxAttempts int) error

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

func (q *Queue) EnqueueJobs(ctx context.Context, jobs []dto.QueuedJob) error {
	if len(jobs) == 0 {
		return nil
	}

	score := float64(time.Now().UnixMilli())
	sm := q.client.B().Zadd().Key(sortedSetKey).Nx().ScoreMember()
	for _, job := range jobs {
		sm = sm.ScoreMember(score, job.URL)
	}
	if err := q.client.Do(ctx, sm.Build()).Error(); err != nil {
		return err
	}

	for _, job := range jobs {
		data, _ := json.Marshal(job)
		_ = q.client.Do(ctx, q.client.B().Hset().Key(payloadKey).FieldValue().FieldValue(job.URL, string(data)).Build()).Error()
	}
	return nil
}

func (q *Queue) Dequeue(ctx context.Context) (dto.QueuedJob, bool, error) {
	scores, err := q.client.Do(ctx, q.client.B().Zpopmin().Key(sortedSetKey).Count(1).Build()).AsZScores()
	if err != nil {
		return dto.QueuedJob{}, false, fmt.Errorf("zpopmin: %w", err)
	}
	if len(scores) == 0 {
		return dto.QueuedJob{}, false, nil
	}

	url := scores[0].Member
	data, err := q.client.Do(ctx, q.client.B().Hget().Key(payloadKey).Field(url).Build()).AsBytes()
	_ = q.client.Do(ctx, q.client.B().Hdel().Key(payloadKey).Field(url).Build()).Error()
	if err != nil {
		return dto.QueuedJob{URL: url}, true, nil
	}

	var job dto.QueuedJob
	if err := json.Unmarshal(data, &job); err != nil {
		return dto.QueuedJob{URL: url}, true, nil
	}
	return job, true, nil
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

func (q *Queue) Nack(ctx context.Context, url string, maxAttempts int) error {
	count, err := q.client.Do(ctx, q.client.B().Hincrby().Key(attemptsKey).Field(url).Increment(1).Build()).AsInt64()
	if err != nil {
		return fmt.Errorf("nack increment: %w", err)
	}
	if int(count) >= maxAttempts {
		score := float64(time.Now().UnixMilli())
		if err := q.client.Do(ctx, q.client.B().Zadd().Key(deadLetterKey).ScoreMember().ScoreMember(score, url).Build()).Error(); err != nil {
			return fmt.Errorf("nack dead letter: %w", err)
		}
		return q.client.Do(ctx, q.client.B().Hdel().Key(attemptsKey).Field(url).Build()).Error()
	}
	backoff := time.Duration(count) * backoffBase
	score := float64(time.Now().Add(backoff).UnixMilli())
	return q.client.Do(ctx, q.client.B().Zadd().Key(sortedSetKey).ScoreMember().ScoreMember(score, url).Build()).Error()
}

// ClearAttempts removes the attempt counter for url. Called on successful processing.
// Not on the JobQueue interface — callers that hold a *Queue can call it directly,
// and worker does a type-assert to reach it.
func (q *Queue) ClearAttempts(ctx context.Context, url string) error {
	return q.client.Do(ctx, q.client.B().Hdel().Key(attemptsKey).Field(url).Build()).Error()
}

func (q *Queue) Close() {
	q.client.Close()
}

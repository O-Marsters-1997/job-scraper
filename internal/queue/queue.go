package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	lastScrapedKeyFmt = "scrape:last:%s"
	backoffBase       = 30 * time.Second
	maxAttempts       = 3
)

type JobQueue interface {
	EnqueueJobs(context.Context, []dto.QueuedJob) error
	EnqueueScrapeRequest(context.Context, dto.ScrapeRequest) error
	ClaimReady(context.Context, Kind, time.Duration) (Item, bool, error)
	Ack(context.Context, Item) error
	Nack(context.Context, Item, string) error
	SetLastScraped(context.Context, string) error
	GetLastScraped(context.Context, string) (time.Time, bool, error)
	Close()
}

type Queue struct{ client valkey.Client }

func New(addr string) (*Queue, error) {
	client, err := valkey.NewClient(valkey.ClientOption{InitAddress: []string{addr}})
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
	for _, job := range jobs {
		payload, err := json.Marshal(job)
		if err != nil {
			return err
		}
		if err := q.Publish(ctx, Detail, job.URL, payload, time.Now()); err != nil {
			return err
		}
	}
	return nil
}

func (q *Queue) EnqueueScrapeRequest(ctx context.Context, req dto.ScrapeRequest) error {
	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if err := q.Publish(ctx, ScrapeRequest, req.Target.ID, payload, time.Now()); err != nil {
		return err
	}
	if err := q.ReplayDeadLetter(ctx, ScrapeRequest, req.Target.ID); err != nil && !errors.Is(err, ErrDeadLetterNotFound) {
		return err
	}
	return nil
}

func (q *Queue) SetLastScraped(ctx context.Context, source string) error {
	key := fmt.Sprintf(lastScrapedKeyFmt, source)
	return q.client.Do(ctx, q.client.B().Set().Key(key).Value(fmt.Sprint(time.Now().UnixMilli())).Build()).Error()
}

func (q *Queue) GetLastScraped(ctx context.Context, source string) (time.Time, bool, error) {
	key := fmt.Sprintf(lastScrapedKeyFmt, source)
	value, err := q.client.Do(ctx, q.client.B().Get().Key(key).Build()).AsInt64()
	if valkey.IsValkeyNil(err) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	return time.UnixMilli(value), true, nil
}

func (q *Queue) DeadLetterCount(ctx context.Context) (int64, error) {
	var total int64
	for _, kind := range []Kind{Detail, ScrapeRequest} {
		keys, _ := queueKeys(kind)
		n, err := q.client.Do(ctx, q.client.B().Zcard().Key(keys[5]).Build()).AsInt64()
		if err != nil {
			return 0, err
		}
		total += n
	}
	return total, nil
}

func (q *Queue) Close() { q.client.Close() }

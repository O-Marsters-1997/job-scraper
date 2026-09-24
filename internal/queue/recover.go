package queue

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/valkey-io/valkey-go"
)

type Kind string

const (
	Detail        Kind = "detail"
	ScrapeRequest Kind = "scrape-request"
)

var ErrLeaseLost = errors.New("queue lease no longer owned")
var ErrDeadLetterNotFound = errors.New("dead letter not found")

type Item struct {
	ID       string
	Kind     Kind
	Payload  []byte
	Token    string
	Attempts int
	Failure  string
}

type Stats struct {
	Ready       int64
	Leased      int64
	Expired     int64
	Dead        int64
	OldestDueAt time.Time
}

func queueKeys(kind Kind) ([]string, error) {
	if kind != Detail && kind != ScrapeRequest {
		return nil, fmt.Errorf("unknown queue kind %q", kind)
	}
	prefix := "queue:" + string(kind) + ":"
	return []string{prefix + "ready", prefix + "leased", prefix + "payload", prefix + "attempts", prefix + "tokens", prefix + "dead", prefix + "failure", prefix + "sequence"}, nil
}

func readyKey(kind Kind) string {
	keys, _ := queueKeys(kind)
	return keys[0]
}

func (q *Queue) eval(ctx context.Context, script string, kind Kind, args ...string) (valkey.ValkeyResult, error) {
	keys, err := queueKeys(kind)
	if err != nil {
		return valkey.ValkeyResult{}, err
	}
	return q.client.Do(ctx, q.client.B().Eval().Script(script).Numkeys(int64(len(keys))).Key(keys...).Arg(args...).Build()), nil
}

const publishScript = `
if redis.call('HSETNX', KEYS[3], ARGV[1], ARGV[2]) == 0 then return 0 end
redis.call('ZADD', KEYS[1], ARGV[3], ARGV[1])
return 1`

func (q *Queue) Publish(ctx context.Context, kind Kind, id string, payload []byte, due time.Time) error {
	if strings.TrimSpace(id) == "" || len(payload) == 0 {
		return errors.New("queue id and payload are required")
	}
	if kind == Detail {
		u, err := url.Parse(id)
		if err != nil || u.Hostname() == "" || u.User != nil || (strings.ToLower(u.Scheme) != "https" && strings.ToLower(u.Scheme) != "http") {
			return errors.New("invalid detail URL")
		}
		u.Scheme = strings.ToLower(u.Scheme)
		u.Host = strings.ToLower(u.Host)
		u.Fragment = ""
		id = u.String()
	}
	result, err := q.eval(ctx, publishScript, kind, id, string(payload), fmt.Sprint(due.UnixMilli()))
	if err != nil {
		return err
	}
	return result.Error()
}

const claimScript = `
local expired = redis.call('ZRANGEBYSCORE', KEYS[2], '-inf', ARGV[1], 'LIMIT', 0, 1)
local id
if #expired > 0 then
  id = expired[1]
  redis.call('ZREM', KEYS[2], id)
  local attempts = redis.call('HINCRBY', KEYS[4], id, 1)
  if attempts >= tonumber(ARGV[3]) then
    redis.call('ZADD', KEYS[6], ARGV[1], id)
    redis.call('HSET', KEYS[7], id, 'lease expired')
    redis.call('HDEL', KEYS[5], id)
    id = nil
  end
end
if not id then
  local ready = redis.call('ZRANGEBYSCORE', KEYS[1], '-inf', ARGV[1], 'LIMIT', 0, 1)
  if #ready == 0 then return {} end
  id = ready[1]
  redis.call('ZREM', KEYS[1], id)
end
local token = tostring(redis.call('INCR', KEYS[8]))
redis.call('HSET', KEYS[5], id, token)
redis.call('ZADD', KEYS[2], tonumber(ARGV[1]) + tonumber(ARGV[2]), id)
return {id, redis.call('HGET', KEYS[3], id), token, tostring(redis.call('HGET', KEYS[4], id) or 0)}`

func (q *Queue) ClaimReady(ctx context.Context, kind Kind, lease time.Duration) (Item, bool, error) {
	if lease <= 0 {
		return Item{}, false, errors.New("lease must be positive")
	}
	result, err := q.eval(ctx, claimScript, kind, fmt.Sprint(time.Now().UnixMilli()), fmt.Sprint(lease.Milliseconds()), fmt.Sprint(maxAttempts))
	if err != nil {
		return Item{}, false, err
	}
	parts, err := result.AsStrSlice()
	if err != nil {
		return Item{}, false, err
	}
	if len(parts) == 0 {
		return Item{}, false, nil
	}
	var attempts int
	_, _ = fmt.Sscan(parts[3], &attempts)
	return Item{ID: parts[0], Kind: kind, Payload: []byte(parts[1]), Token: parts[2], Attempts: attempts}, true, nil
}

const ackScript = `
if redis.call('HGET', KEYS[5], ARGV[1]) ~= ARGV[2] then return 0 end
redis.call('ZREM', KEYS[2], ARGV[1])
redis.call('HDEL', KEYS[3], ARGV[1])
redis.call('HDEL', KEYS[4], ARGV[1])
redis.call('HDEL', KEYS[5], ARGV[1])
redis.call('HDEL', KEYS[7], ARGV[1])
return 1`

func (q *Queue) Ack(ctx context.Context, item Item) error {
	result, err := q.eval(ctx, ackScript, item.Kind, item.ID, item.Token)
	if err != nil {
		return err
	}
	return transitionResult(result, ErrLeaseLost)
}

const nackScript = `
if redis.call('HGET', KEYS[5], ARGV[1]) ~= ARGV[2] then return 0 end
redis.call('ZREM', KEYS[2], ARGV[1])
redis.call('HDEL', KEYS[5], ARGV[1])
local attempts = redis.call('HINCRBY', KEYS[4], ARGV[1], 1)
redis.call('HSET', KEYS[7], ARGV[1], ARGV[4])
if attempts >= tonumber(ARGV[5]) then
  redis.call('ZADD', KEYS[6], ARGV[3], ARGV[1])
else
  redis.call('ZADD', KEYS[1], tonumber(ARGV[3]) + tonumber(ARGV[6]) * (2 ^ (attempts - 1)), ARGV[1])
end
return 1`

func (q *Queue) Nack(ctx context.Context, item Item, failure string) error {
	result, err := q.eval(ctx, nackScript, item.Kind, item.ID, item.Token, fmt.Sprint(time.Now().UnixMilli()), failure, fmt.Sprint(maxAttempts), fmt.Sprint(backoffBase.Milliseconds()))
	if err != nil {
		return err
	}
	return transitionResult(result, ErrLeaseLost)
}

func transitionResult(result valkey.ValkeyResult, missing error) error {
	n, err := result.AsInt64()
	if err != nil {
		return err
	}
	if n == 0 {
		return missing
	}
	return nil
}

const replayScript = `
if redis.call('ZREM', KEYS[6], ARGV[1]) == 0 then return 0 end
redis.call('HDEL', KEYS[4], ARGV[1])
redis.call('HDEL', KEYS[7], ARGV[1])
redis.call('ZADD', KEYS[1], ARGV[2], ARGV[1])
return 1`

func (q *Queue) ReplayDeadLetter(ctx context.Context, kind Kind, id string) error {
	result, err := q.eval(ctx, replayScript, kind, id, fmt.Sprint(time.Now().UnixMilli()))
	if err != nil {
		return err
	}
	return transitionResult(result, ErrDeadLetterNotFound)
}

func (q *Queue) DeadLetters(ctx context.Context, kind Kind) ([]Item, error) {
	keys, err := queueKeys(kind)
	if err != nil {
		return nil, err
	}
	ids, err := q.client.Do(ctx, q.client.B().Zrange().Key(keys[5]).Min("0").Max("-1").Build()).AsStrSlice()
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(ids))
	for _, id := range ids {
		payload, err := q.client.Do(ctx, q.client.B().Hget().Key(keys[2]).Field(id).Build()).AsBytes()
		if err != nil {
			return nil, err
		}
		failure, err := q.client.Do(ctx, q.client.B().Hget().Key(keys[6]).Field(id).Build()).ToString()
		if err != nil {
			return nil, err
		}
		items = append(items, Item{ID: id, Kind: kind, Payload: payload, Failure: failure})
	}
	return items, nil
}

func (q *Queue) Stats(ctx context.Context, kind Kind) (Stats, error) {
	keys, err := queueKeys(kind)
	if err != nil {
		return Stats{}, err
	}
	var stats Stats
	for _, entry := range []struct {
		key string
		out *int64
	}{{keys[0], &stats.Ready}, {keys[1], &stats.Leased}, {keys[5], &stats.Dead}} {
		*entry.out, err = q.client.Do(ctx, q.client.B().Zcard().Key(entry.key).Build()).AsInt64()
		if err != nil {
			return Stats{}, err
		}
	}
	stats.Expired, err = q.client.Do(ctx, q.client.B().Zcount().Key(keys[1]).Min("-inf").Max(fmt.Sprint(time.Now().UnixMilli())).Build()).AsInt64()
	if err != nil {
		return Stats{}, err
	}
	if stats.Ready > 0 {
		entries, err := q.client.Do(ctx, q.client.B().Zrange().Key(keys[0]).Min("0").Max("0").Withscores().Build()).AsZScores()
		if err != nil {
			return Stats{}, err
		}
		stats.OldestDueAt = time.UnixMilli(int64(entries[0].Score))
	}
	return stats, nil
}

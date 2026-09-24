package queue

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"
)

type SourceKind string

const (
	SourceListing SourceKind = "listing"
	SourceDetail  SourceKind = "detail"
)

type SourceItem struct {
	ID       string
	Source   string
	Kind     SourceKind
	Payload  []byte
	Token    string
	Attempts int
}

var sourceName = regexp.MustCompile(`^[a-z0-9_-]+$`)

var sourceKeys = []string{
	"queue:source:sources", "queue:source:payload", "queue:source:task-source",
	"queue:source:task-kind", "queue:source:attempts", "queue:source:tokens",
	"queue:source:active", "queue:source:leases", "queue:source:gaps",
	"queue:source:pauses", "queue:source:failures", "queue:source:dead",
	"queue:source:sequence", "queue:source:cooldowns",
}

func (q *Queue) sourceEval(ctx context.Context, script string, args ...string) ([]string, error) {
	return q.client.Do(ctx, q.client.B().Eval().Script(script).Numkeys(int64(len(sourceKeys))).Key(sourceKeys...).Arg(args...).Build()).AsStrSlice()
}

const sourcePublishScript = `
local id, source, kind, payload, due, gap = ARGV[1], ARGV[2], ARGV[3], ARGV[4], tonumber(ARGV[5]), tonumber(ARGV[6])
if redis.call('HSETNX', KEYS[2], id, payload) == 0 then return {'0'} end
redis.call('HSET', KEYS[3], id, source)
redis.call('HSET', KEYS[4], id, kind)
redis.call('HSET', KEYS[9], source, gap)
redis.call('ZADD', 'queue:source:ready:' .. source .. ':' .. kind, due, id)
local eligible = math.max(due, tonumber(redis.call('HGET', KEYS[10], source) or 0), tonumber(redis.call('HGET', KEYS[14], source) or 0), tonumber(redis.call('ZSCORE', KEYS[8], source) or 0))
local current = redis.call('ZSCORE', KEYS[1], source)
if not current or eligible < tonumber(current) then redis.call('ZADD', KEYS[1], eligible, source) end
return {'1'}`

func (q *Queue) PublishSource(ctx context.Context, source string, kind SourceKind, id string, payload []byte, due time.Time, gap time.Duration) error {
	if !sourceName.MatchString(source) || (kind != SourceListing && kind != SourceDetail) || id == "" || len(payload) == 0 || gap < 0 {
		return errors.New("invalid source task")
	}
	_, err := q.sourceEval(ctx, sourcePublishScript, id, source, string(kind), string(payload), fmt.Sprint(due.UnixMilli()), fmt.Sprint(gap.Milliseconds()))
	return err
}

const sourceClaimScript = `
local now, lease, maxAttempts, backoff = tonumber(ARGV[1]), tonumber(ARGV[2]), tonumber(ARGV[3]), tonumber(ARGV[4])
local function ready(source, kind)
  local key = 'queue:source:ready:' .. source .. ':' .. kind
  local head = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
  if #head == 0 then return nil, nil end
  return head[1], tonumber(head[2])
end
local function nextDue(source)
  local _, listing = ready(source, 'listing')
  local _, detail = ready(source, 'detail')
  if not listing then return detail end
  if not detail then return listing end
  return math.min(listing, detail)
end
local sources = redis.call('ZRANGEBYSCORE', KEYS[1], '-inf', now, 'LIMIT', 0, 64)
for _, source in ipairs(sources) do
  local active = redis.call('HGET', KEYS[7], source)
  local expiry = tonumber(redis.call('ZSCORE', KEYS[8], source) or 0)
  if active and expiry > now then
    redis.call('ZADD', KEYS[1], expiry, source)
  else
    if active then
      redis.call('HDEL', KEYS[7], source)
      redis.call('ZREM', KEYS[8], source)
      redis.call('HDEL', KEYS[6], active)
      local attempts = redis.call('HINCRBY', KEYS[5], active, 1)
      if attempts >= maxAttempts then
        redis.call('ZADD', KEYS[12], now, active)
        redis.call('HSET', KEYS[11], active, 'lease expired')
      else
        local kind = redis.call('HGET', KEYS[4], active)
        redis.call('ZADD', 'queue:source:ready:' .. source .. ':' .. kind, now + backoff * (2 ^ (attempts - 1)), active)
      end
    end
    local listing, listingDue = ready(source, 'listing')
    local detail, detailDue = ready(source, 'detail')
    local due = nextDue(source)
    if not due then
      redis.call('ZREM', KEYS[1], source)
    else
      local eligible = math.max(due, tonumber(redis.call('HGET', KEYS[10], source) or 0), tonumber(redis.call('HGET', KEYS[14], source) or 0))
      if eligible > now then
        redis.call('ZADD', KEYS[1], eligible, source)
      else
        local id, kind
        if listing and listingDue <= now then id, kind = listing, 'listing'
        elseif detail and detailDue <= now then id, kind = detail, 'detail' end
        if id then
          redis.call('ZREM', 'queue:source:ready:' .. source .. ':' .. kind, id)
          local token = tostring(redis.call('INCR', KEYS[13]))
          redis.call('HSET', KEYS[6], id, token)
          redis.call('HSET', KEYS[7], source, id)
          redis.call('ZADD', KEYS[8], now + lease, source)
          redis.call('ZADD', KEYS[1], now + lease, source)
          return {id, source, kind, redis.call('HGET', KEYS[2], id), token, tostring(redis.call('HGET', KEYS[5], id) or 0)}
        end
      end
    end
  end
end
return {}`

func (q *Queue) ClaimSource(ctx context.Context, lease time.Duration) (SourceItem, bool, error) {
	if lease < time.Millisecond {
		return SourceItem{}, false, errors.New("lease must be at least one millisecond")
	}
	parts, err := q.sourceEval(ctx, sourceClaimScript, fmt.Sprint(time.Now().UnixMilli()), fmt.Sprint(lease.Milliseconds()), fmt.Sprint(maxAttempts), fmt.Sprint(backoffBase.Milliseconds()))
	if err != nil || len(parts) == 0 {
		return SourceItem{}, false, err
	}
	var attempts int
	_, _ = fmt.Sscan(parts[5], &attempts)
	return SourceItem{ID: parts[0], Source: parts[1], Kind: SourceKind(parts[2]), Payload: []byte(parts[3]), Token: parts[4], Attempts: attempts}, true, nil
}

const sourceFinishScript = `
local id, source, token, now, failure, maxAttempts, backoff = ARGV[1], ARGV[2], ARGV[3], tonumber(ARGV[4]), ARGV[5], tonumber(ARGV[6]), tonumber(ARGV[7])
if redis.call('HGET', KEYS[7], source) ~= id or redis.call('HGET', KEYS[6], id) ~= token or tonumber(redis.call('ZSCORE', KEYS[8], source) or 0) < now then return {'0'} end
redis.call('HDEL', KEYS[7], source)
redis.call('ZREM', KEYS[8], source)
redis.call('HDEL', KEYS[6], id)
if failure == '' then
  redis.call('HDEL', KEYS[2], id)
  redis.call('HDEL', KEYS[3], id)
  redis.call('HDEL', KEYS[4], id)
  redis.call('HDEL', KEYS[5], id)
  redis.call('HDEL', KEYS[11], id)
else
  local attempts = redis.call('HINCRBY', KEYS[5], id, 1)
  redis.call('HSET', KEYS[11], id, failure)
  if attempts >= maxAttempts then
    redis.call('ZADD', KEYS[12], now, id)
  else
    local kind = redis.call('HGET', KEYS[4], id)
    redis.call('ZADD', 'queue:source:ready:' .. source .. ':' .. kind, now + backoff * (2 ^ (attempts - 1)), id)
  end
end
local cooldown = now + tonumber(redis.call('HGET', KEYS[9], source) or 0)
redis.call('HSET', KEYS[14], source, cooldown)
local listing = redis.call('ZRANGE', 'queue:source:ready:' .. source .. ':listing', 0, 0, 'WITHSCORES')
local detail = redis.call('ZRANGE', 'queue:source:ready:' .. source .. ':detail', 0, 0, 'WITHSCORES')
local due
if #listing > 0 then due = tonumber(listing[2]) end
if #detail > 0 and (not due or tonumber(detail[2]) < due) then due = tonumber(detail[2]) end
if due then
  redis.call('ZADD', KEYS[1], math.max(due, cooldown, tonumber(redis.call('HGET', KEYS[10], source) or 0)), source)
else
  redis.call('ZREM', KEYS[1], source)
end
return {'1'}`

func (q *Queue) finishSource(ctx context.Context, item SourceItem, failure string) error {
	parts, err := q.sourceEval(ctx, sourceFinishScript, item.ID, item.Source, item.Token, fmt.Sprint(time.Now().UnixMilli()), failure, fmt.Sprint(maxAttempts), fmt.Sprint(backoffBase.Milliseconds()))
	if err != nil {
		return err
	}
	if len(parts) == 0 || parts[0] != "1" {
		return ErrLeaseLost
	}
	return nil
}

func (q *Queue) AckSource(ctx context.Context, item SourceItem) error {
	return q.finishSource(ctx, item, "")
}

func (q *Queue) NackSource(ctx context.Context, item SourceItem, failure string) error {
	if failure == "" {
		return errors.New("failure reason is required")
	}
	return q.finishSource(ctx, item, failure)
}

const sourceRenewScript = `
if redis.call('HGET', KEYS[7], ARGV[1]) ~= ARGV[2] or redis.call('HGET', KEYS[6], ARGV[2]) ~= ARGV[3] or tonumber(redis.call('ZSCORE', KEYS[8], ARGV[1]) or 0) < tonumber(ARGV[5]) then return {'0'} end
redis.call('ZADD', KEYS[8], ARGV[4], ARGV[1])
redis.call('ZADD', KEYS[1], math.max(tonumber(ARGV[4]), tonumber(redis.call('HGET', KEYS[10], ARGV[1]) or 0)), ARGV[1])
return {'1'}`

func (q *Queue) RenewSource(ctx context.Context, item SourceItem, lease time.Duration) error {
	if lease < time.Millisecond {
		return errors.New("lease must be at least one millisecond")
	}
	now := time.Now()
	parts, err := q.sourceEval(ctx, sourceRenewScript, item.Source, item.ID, item.Token, fmt.Sprint(now.Add(lease).UnixMilli()), fmt.Sprint(now.UnixMilli()))
	if err != nil {
		return err
	}
	if len(parts) == 0 || parts[0] != "1" {
		return ErrLeaseLost
	}
	return nil
}

const sourcePauseScript = `
redis.call('HSET', KEYS[10], ARGV[1], ARGV[2])
local listing = redis.call('ZRANGE', 'queue:source:ready:' .. ARGV[1] .. ':listing', 0, 0, 'WITHSCORES')
local detail = redis.call('ZRANGE', 'queue:source:ready:' .. ARGV[1] .. ':detail', 0, 0, 'WITHSCORES')
local due = tonumber(redis.call('ZSCORE', KEYS[8], ARGV[1]) or 0)
if #listing > 0 and (due == 0 or tonumber(listing[2]) < due) then due = tonumber(listing[2]) end
if #detail > 0 and (due == 0 or tonumber(detail[2]) < due) then due = tonumber(detail[2]) end
if due > 0 then
  redis.call('ZADD', KEYS[1], math.max(due, tonumber(ARGV[2]), tonumber(redis.call('HGET', KEYS[14], ARGV[1]) or 0), tonumber(redis.call('ZSCORE', KEYS[8], ARGV[1]) or 0)), ARGV[1])
else
  redis.call('ZREM', KEYS[1], ARGV[1])
end
return {'1'}`

func (q *Queue) PauseSource(ctx context.Context, source string, until time.Time) error {
	if !sourceName.MatchString(source) {
		return errors.New("invalid source")
	}
	_, err := q.sourceEval(ctx, sourcePauseScript, source, fmt.Sprint(until.UnixMilli()))
	return err
}

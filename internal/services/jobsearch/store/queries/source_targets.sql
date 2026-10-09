-- name: ListSourceTargetsByUser :many
SELECT * FROM source_targets WHERE user_id = $1 ORDER BY source, value;

-- name: CreateSourceTarget :one
INSERT INTO source_targets (user_id, source, value, enabled, filters, interval_minutes, weekdays, window_start, window_end, timezone, next_run_at)
VALUES ($1, $2, $3, $4, $5, sqlc.narg('interval_minutes'),
        COALESCE(sqlc.narg('weekdays')::smallint, 31),
        COALESCE(sqlc.narg('window_start')::time, '08:00'),
        COALESCE(sqlc.narg('window_end')::time, '18:00'),
        COALESCE(sqlc.narg('timezone')::text, 'Europe/London'),
        sqlc.narg('next_run_at'))
RETURNING *;

-- name: CreateSourceTargetWithRun :one
INSERT INTO source_targets (user_id, source, value, enabled, filters, interval_minutes, weekdays, window_start, window_end, timezone, next_run_at, run_id, run_status)
VALUES ($1, $2, $3, $4, $5, sqlc.narg('interval_minutes'),
        COALESCE(sqlc.narg('weekdays')::smallint, 31),
        COALESCE(sqlc.narg('window_start')::time, '08:00'),
        COALESCE(sqlc.narg('window_end')::time, '18:00'),
        COALESCE(sqlc.narg('timezone')::text, 'Europe/London'),
        sqlc.narg('next_run_at'), gen_random_uuid(), 'queued')
RETURNING *;

-- name: UpdateSourceTarget :one
UPDATE source_targets SET
    enabled                = COALESCE(sqlc.narg('enabled'), enabled),
    disabled_reason        = CASE WHEN sqlc.narg('enabled')::boolean THEN '' ELSE disabled_reason END,
    interval_minutes       = CASE WHEN sqlc.arg('set_window')::boolean THEN sqlc.narg('interval_minutes') ELSE interval_minutes END,
    weekdays               = COALESCE(sqlc.narg('weekdays')::smallint, weekdays),
    window_start           = COALESCE(sqlc.narg('window_start')::time, window_start),
    window_end             = COALESCE(sqlc.narg('window_end')::time, window_end),
    timezone               = COALESCE(sqlc.narg('timezone')::text, timezone),
    next_run_at            = CASE WHEN sqlc.arg('set_window')::boolean THEN sqlc.narg('next_run_at') ELSE next_run_at END,
    updated_at             = NOW()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DisableSourceTargets :execrows
UPDATE source_targets SET enabled = FALSE, disabled_reason = $2,
    run_status = CASE WHEN run_status IN ('queued', 'running') THEN 'failed' ELSE run_status END,
    last_run_error = CASE WHEN run_status IN ('queued', 'running') THEN $2 ELSE last_run_error END,
    updated_at = NOW()
WHERE source = $1 AND enabled = TRUE;

-- name: DeleteSourceTarget :exec
DELETE FROM source_targets WHERE id = $1 AND user_id = $2;

-- name: StartSourceTargetRun :one
UPDATE source_targets SET run_id = gen_random_uuid(), run_status = 'queued',
    next_run_at = sqlc.narg('next_run_at'),
    enabled = TRUE, disabled_reason = '', last_run_error = '', updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: GetSourceTarget :one
SELECT * FROM source_targets WHERE id = $1;

-- name: TransitionSourceTargetRun :one
UPDATE source_targets SET run_status = $3, last_run_error = $4,
    last_run_at = CASE WHEN $3 IN ('succeeded', 'failed') THEN NOW() ELSE last_run_at END,
    last_succeeded_at = CASE WHEN $3 = 'succeeded' THEN NOW() ELSE last_succeeded_at END,
    updated_at = NOW()
WHERE id = $1 AND run_id = $2
RETURNING *;

-- name: ListRecoverableSourceTargets :many
SELECT * FROM source_targets
WHERE run_id IS NOT NULL AND enabled = TRUE
  AND (run_status = 'queued' AND updated_at < NOW() - INTERVAL '1 minute'
       OR run_status = 'running' AND updated_at < NOW() - INTERVAL '30 minutes')
ORDER BY updated_at;

-- name: ClaimRecoverableSourceTarget :one
UPDATE source_targets SET updated_at = NOW()
WHERE id = $1 AND run_id = $2 AND enabled = TRUE
  AND (run_status = 'queued' AND updated_at < NOW() - INTERVAL '1 minute'
       OR run_status = 'running' AND updated_at < NOW() - INTERVAL '30 minutes')
RETURNING *;

-- name: LockDueSourceTargets :many
SELECT * FROM source_targets
WHERE enabled = TRUE AND interval_minutes IS NOT NULL AND next_run_at <= NOW()
  AND run_status NOT IN ('queued', 'running')
ORDER BY next_run_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

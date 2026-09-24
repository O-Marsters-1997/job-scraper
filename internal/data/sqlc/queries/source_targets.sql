-- name: ListSourceTargetsByUser :many
SELECT * FROM source_targets WHERE user_id = $1 ORDER BY source, value;

-- name: ListEnabledSourceTargets :many
SELECT * FROM source_targets WHERE enabled = TRUE ORDER BY source, value;

-- name: ListDueSourceTargets :many
SELECT * FROM source_targets
WHERE enabled = TRUE
  AND (last_checked_at IS NULL
       OR last_checked_at <= NOW() - make_interval(mins => check_interval_minutes))
ORDER BY source, value;

-- name: TouchSourceTargetsChecked :exec
UPDATE source_targets SET last_checked_at = NOW()
WHERE source = $1 AND value = $2 AND enabled = TRUE;

-- name: CreateSourceTarget :one
INSERT INTO source_targets (user_id, source, value, enabled, filters)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CreateSourceTargetWithRun :one
INSERT INTO source_targets (user_id, source, value, enabled, filters, run_id, run_status)
VALUES ($1, $2, $3, $4, $5, gen_random_uuid(), 'queued')
RETURNING *;

-- name: UpsertSourceTargetForCompany :one
INSERT INTO source_targets (user_id, source, value, enabled, filters, company_id, check_interval_minutes)
VALUES ($1, $2, $3, $4, '{}', $5, COALESCE(NULLIF(sqlc.arg(check_interval_minutes)::int, 0), 360))
ON CONFLICT (user_id, source, value, filters) DO UPDATE SET
    enabled    = EXCLUDED.enabled,
    company_id = EXCLUDED.company_id,
    check_interval_minutes = COALESCE(NULLIF(sqlc.arg(check_interval_minutes)::int, 0), source_targets.check_interval_minutes),
    updated_at = NOW()
RETURNING *;

-- name: UpdateSourceTarget :one
UPDATE source_targets SET
    enabled                = COALESCE(sqlc.narg('enabled'), enabled),
    check_interval_minutes = COALESCE(sqlc.narg('check_interval_minutes'), check_interval_minutes),
    updated_at             = NOW()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: ListUserIDsForTarget :many
SELECT DISTINCT user_id FROM source_targets
WHERE enabled = TRUE
  AND source = $1
  AND (sqlc.arg(value)::text = '' OR value = sqlc.arg(value));

-- name: DeleteSourceTarget :exec
DELETE FROM source_targets WHERE id = $1 AND user_id = $2;

-- name: SetSourceTargetRunState :one
UPDATE source_targets SET
    run_status = $2,
    last_run_error = $3,
    last_run_at = CASE WHEN $2 IN ('succeeded', 'failed') THEN NOW() ELSE last_run_at END,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: StartSourceTargetRun :one
UPDATE source_targets SET run_id = gen_random_uuid(), run_status = 'queued',
    enabled = TRUE, last_run_error = '', updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: TransitionSourceTargetRun :one
UPDATE source_targets SET run_status = $3, last_run_error = $4,
    last_run_at = CASE WHEN $3 IN ('succeeded', 'failed') THEN NOW() ELSE last_run_at END,
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

-- name: GetSourceTarget :one
SELECT * FROM source_targets WHERE id = $1;

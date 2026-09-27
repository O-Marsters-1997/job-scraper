-- name: OpsState :one
SELECT
    (SELECT count(*) FROM effect_outbox WHERE status IN ('pending', 'running')) AS outbox_pending,
    (SELECT MIN(created_at)::timestamptz FROM effect_outbox WHERE status IN ('pending', 'running')) AS oldest_pending_created_at,
    (SELECT count(*) FROM effect_outbox WHERE status = 'failed') AS outbox_failed;

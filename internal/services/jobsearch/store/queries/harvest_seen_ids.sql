-- name: ListHarvestSeenIDs :many
SELECT external_id FROM harvest_seen_ids WHERE source = $1;

-- name: RecordHarvestSeenIDs :exec
INSERT INTO harvest_seen_ids (source, external_id, outcome)
SELECT @source::text, unnest(@external_ids::text[]), @outcome::text
ON CONFLICT (source, external_id) DO UPDATE SET outcome = EXCLUDED.outcome;

-- name: DeleteHarvestSeenIDs :exec
DELETE FROM harvest_seen_ids WHERE source = @source AND external_id = ANY(@external_ids::text[]);

-- name: CloseJobsByPostingID :exec
WITH closed AS (
    UPDATE jobs SET closed_at = NOW()
    WHERE source = @source AND provider_posting_id = ANY(@posting_ids::text[]) AND closed_at IS NULL
    RETURNING id
)
DELETE FROM option_answers WHERE job_id IN (SELECT id FROM closed);

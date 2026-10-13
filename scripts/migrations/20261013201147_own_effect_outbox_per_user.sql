-- +goose Up
ALTER TABLE effect_outbox ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE;
DROP INDEX effect_outbox_pending_idx;

INSERT INTO effect_outbox (job_id, user_id, fingerprint, model, first_discovery, status, attempts, due_at, lease_until, last_error, created_at)
SELECT e.job_id, i.user_id, e.fingerprint, e.model, e.first_discovery, e.status, e.attempts, e.due_at, e.lease_until, e.last_error, e.created_at
FROM effect_outbox e
JOIN jobs j ON j.id = e.job_id
JOIN LATERAL (
    SELECT tc.user_id FROM tracked_companies tc JOIN companies c ON c.id = tc.company_id
    WHERE tc.enabled AND (c.id = j.company_id OR c.slug = j.company_slug)
    UNION
    SELECT st.user_id FROM source_targets st WHERE st.enabled AND st.source = j.source
) i ON TRUE
WHERE e.user_id IS NULL AND e.status IN ('pending', 'running');

DELETE FROM effect_outbox WHERE user_id IS NULL;

ALTER TABLE effect_outbox ALTER COLUMN user_id SET NOT NULL;
CREATE UNIQUE INDEX effect_outbox_pending_idx ON effect_outbox (job_id, user_id, fingerprint, model)
    WHERE status IN ('pending', 'running');
CREATE INDEX effect_outbox_user_id_idx ON effect_outbox (user_id);

-- +goose Down
DROP INDEX effect_outbox_user_id_idx;
DROP INDEX effect_outbox_pending_idx;

DELETE FROM effect_outbox a
USING effect_outbox b
WHERE a.status IN ('pending', 'running') AND b.status IN ('pending', 'running')
    AND a.job_id = b.job_id AND a.fingerprint = b.fingerprint AND a.model = b.model
    AND a.id > b.id;

ALTER TABLE effect_outbox DROP COLUMN user_id;
CREATE UNIQUE INDEX effect_outbox_pending_idx ON effect_outbox (job_id, fingerprint, model)
    WHERE status IN ('pending', 'running');

-- +goose Up
WITH merged AS (
    SELECT (array_agg(id ORDER BY created_at, id))[1] AS keep_id,
           left(string_agg(body, E'\n\n' ORDER BY created_at, id), 10000) AS body,
           max(updated_at) AS updated_at
    FROM job_notes
    GROUP BY user_id, job_id
    HAVING count(*) > 1
)
UPDATE job_notes n
SET body = m.body, updated_at = m.updated_at
FROM merged m
WHERE n.id = m.keep_id;

DELETE FROM job_notes n
USING (
    SELECT id, row_number() OVER (PARTITION BY user_id, job_id ORDER BY created_at, id) AS rn
    FROM job_notes
) r
WHERE n.id = r.id AND r.rn > 1;

ALTER TABLE job_notes
    DROP CONSTRAINT job_notes_body_check,
    ADD CONSTRAINT job_notes_body_check CHECK (length(body) <= 10000),
    ADD CONSTRAINT job_notes_user_job_key UNIQUE (user_id, job_id);
DROP INDEX job_notes_user_job_idx;
CREATE INDEX job_notes_job_idx ON job_notes (job_id);

-- +goose Down
DROP INDEX job_notes_job_idx;
CREATE INDEX job_notes_user_job_idx ON job_notes (user_id, job_id, created_at DESC);
DELETE FROM job_notes WHERE body = '';
ALTER TABLE job_notes
    DROP CONSTRAINT job_notes_user_job_key,
    DROP CONSTRAINT job_notes_body_check,
    ADD CONSTRAINT job_notes_body_check CHECK (length(body) BETWEEN 1 AND 10000);

-- +goose Up
INSERT INTO job_notes (user_id, job_id, body, created_at, updated_at)
SELECT user_id, job_id, left(notes, 10000), created_at, updated_at
FROM applications
WHERE notes ~ '\S';

ALTER TABLE applications DROP COLUMN notes;

-- +goose Down
ALTER TABLE applications ADD COLUMN notes TEXT;

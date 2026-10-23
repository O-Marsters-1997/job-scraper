-- +goose Up
UPDATE tailored_cvs t SET outcome = 'discarded', draft_doc_id = NULL
WHERE t.outcome IS NULL
  AND EXISTS (
    SELECT 1 FROM tailored_cvs n
    WHERE n.user_id = t.user_id AND n.job_id = t.job_id AND n.outcome IS NULL
      AND (n.created_at, n.id) > (t.created_at, t.id)
  );
ALTER TABLE tailored_cvs ADD COLUMN mode TEXT NOT NULL DEFAULT 'generate' CONSTRAINT tailored_cvs_mode_check CHECK (mode IN ('as_is', 'generate'));
CREATE UNIQUE INDEX tailored_cvs_live_idx ON tailored_cvs (user_id, job_id) WHERE outcome IS NULL;

-- +goose Down
DROP INDEX tailored_cvs_live_idx;
ALTER TABLE tailored_cvs DROP COLUMN mode;

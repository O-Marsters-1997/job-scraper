-- +goose Up
ALTER TABLE tailored_cvs DROP CONSTRAINT tailored_cvs_status_check;
ALTER TABLE tailored_cvs ADD CONSTRAINT tailored_cvs_status_check
    CHECK (status IN ('pending', 'running', 'keeping', 'ready', 'failed'));
ALTER TABLE tailored_cvs ADD COLUMN kept_as TEXT CHECK (kept_as IN ('tab', 'doc'));
ALTER TABLE tailored_cvs ADD COLUMN keep_note TEXT NOT NULL DEFAULT '';
DROP INDEX tailored_cvs_claim_idx;
CREATE INDEX tailored_cvs_claim_idx ON tailored_cvs (due_at, id) WHERE status IN ('pending', 'running', 'keeping');

-- +goose Down
DROP INDEX tailored_cvs_claim_idx;
CREATE INDEX tailored_cvs_claim_idx ON tailored_cvs (due_at, id) WHERE status IN ('pending', 'running');
ALTER TABLE tailored_cvs DROP COLUMN keep_note;
ALTER TABLE tailored_cvs DROP COLUMN kept_as;
UPDATE tailored_cvs SET status = 'ready' WHERE status = 'keeping';
ALTER TABLE tailored_cvs DROP CONSTRAINT tailored_cvs_status_check;
ALTER TABLE tailored_cvs ADD CONSTRAINT tailored_cvs_status_check
    CHECK (status IN ('pending', 'running', 'ready', 'failed'));

-- +goose Up
CREATE TABLE tailored_cvs (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id          UUID        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    base_doc_id     TEXT        NOT NULL,
    base_tab_id     TEXT        NOT NULL,
    achievement_ids UUID[]      NOT NULL,
    edit_set        JSONB,
    findings        JSONB       NOT NULL DEFAULT '[]',
    raw_output      TEXT        NOT NULL DEFAULT '',
    model           TEXT        NOT NULL DEFAULT '',
    prompt_version  TEXT        NOT NULL DEFAULT '',
    job_fingerprint TEXT        NOT NULL DEFAULT '',
    cost            REAL        NOT NULL DEFAULT 0,
    draft_doc_id    TEXT,
    status          TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'ready', 'failed')),
    outcome         TEXT        CHECK (outcome IN ('kept', 'discarded')),
    attempts        INT         NOT NULL DEFAULT 0,
    due_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    lease_until     TIMESTAMPTZ,
    last_error      TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX tailored_cvs_kept_idx ON tailored_cvs (user_id, job_id) WHERE outcome = 'kept';
CREATE INDEX tailored_cvs_claim_idx ON tailored_cvs (due_at, id) WHERE status IN ('pending', 'running');

-- +goose Down
DROP TABLE tailored_cvs;

-- +goose Up
CREATE TABLE profile_attempts (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    draft_id   UUID NOT NULL REFERENCES tailored_cvs(id) ON DELETE CASCADE,
    text       TEXT NOT NULL,
    steer      TEXT NOT NULL DEFAULT '',
    rejection  TEXT NOT NULL DEFAULT '',
    prompt_key TEXT NOT NULL,
    model      TEXT NOT NULL,
    cost       NUMERIC(10, 6) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX profile_attempts_draft_id_created_at_idx ON profile_attempts (draft_id, created_at);

-- +goose Down
DROP TABLE profile_attempts;

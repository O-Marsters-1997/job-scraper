-- +goose Up
CREATE TABLE preference_labels (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id         UUID        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    draft_id       UUID        REFERENCES tailored_cvs(id) ON DELETE SET NULL,
    kind           TEXT        NOT NULL CHECK (kind IN ('bullet')),
    achievement_id UUID        REFERENCES achievements(id) ON DELETE SET NULL,
    p_yes          DOUBLE PRECISION,
    p_no           DOUBLE PRECISION,
    p_not_stated   DOUBLE PRECISION,
    confidence     DOUBLE PRECISION,
    preselected    BOOLEAN     NOT NULL,
    kept           BOOLEAN     NOT NULL,
    position       INT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX preference_labels_user_job_idx ON preference_labels (user_id, job_id);

-- +goose Down
DROP TABLE preference_labels;

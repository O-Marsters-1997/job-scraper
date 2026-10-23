-- +goose Up
CREATE TABLE achievement_outcomes (
    draft_id                    UUID        NOT NULL REFERENCES tailored_cvs(id) ON DELETE CASCADE,
    achievement_id              UUID        NOT NULL REFERENCES achievements(id) ON DELETE CASCADE,
    user_id                     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id                      UUID        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    p_yes                       DOUBLE PRECISION,
    p_no                        DOUBLE PRECISION,
    on_source_cv                BOOLEAN     NOT NULL,
    in_final_cv                 BOOLEAN     NOT NULL,
    outscored_kept_but_left_out BOOLEAN     NOT NULL,
    reworded                    BOOLEAN     NOT NULL,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (draft_id, achievement_id)
);
CREATE INDEX achievement_outcomes_achievement_idx ON achievement_outcomes (achievement_id);
DROP TABLE preference_labels;

-- +goose Down
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
CREATE INDEX preference_labels_draft_idx ON preference_labels (draft_id);
DROP TABLE achievement_outcomes;

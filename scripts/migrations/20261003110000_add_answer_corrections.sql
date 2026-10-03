-- +goose Up
CREATE TABLE answer_corrections (
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id     UUID        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    option_id  TEXT        NOT NULL REFERENCES scoring_options(id),
    value      TEXT        NOT NULL CHECK (value IN ('yes', 'no')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, job_id, option_id)
);

-- +goose Down
DROP TABLE answer_corrections;

-- +goose Up
CREATE TYPE scoring_dimension AS ENUM ('tech', 'role', 'domain', 'seniority', 'work', 'stage');

CREATE TABLE scoring_options (
    id         TEXT PRIMARY KEY,
    dimension  scoring_dimension NOT NULL,
    label      TEXT NOT NULL,
    question   TEXT NOT NULL,
    retired_at TIMESTAMPTZ
);

-- +goose Down
DROP TABLE scoring_options;
DROP TYPE scoring_dimension;

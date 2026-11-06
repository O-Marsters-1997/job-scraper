-- +goose Up
CREATE TABLE prompt_part_overrides (
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    generation   TEXT        NOT NULL,
    path         TEXT        NOT NULL,
    text         TEXT        NOT NULL
        CONSTRAINT prompt_part_overrides_text_check CHECK (length(text) BETWEEN 1 AND 20000),
    base_default TEXT        NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, generation, path)
);

-- +goose Down
DROP TABLE prompt_part_overrides;

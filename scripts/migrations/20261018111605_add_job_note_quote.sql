-- +goose Up
ALTER TABLE job_notes
    ADD COLUMN quote        TEXT
        CONSTRAINT job_notes_quote_check CHECK (length(quote) BETWEEN 1 AND 2000),
    ADD COLUMN quote_prefix TEXT NOT NULL DEFAULT ''
        CONSTRAINT job_notes_quote_prefix_check CHECK (length(quote_prefix) <= 64),
    ADD COLUMN quote_suffix TEXT NOT NULL DEFAULT ''
        CONSTRAINT job_notes_quote_suffix_check CHECK (length(quote_suffix) <= 64);

-- +goose Down
ALTER TABLE job_notes
    DROP COLUMN quote_suffix,
    DROP COLUMN quote_prefix,
    DROP COLUMN quote;

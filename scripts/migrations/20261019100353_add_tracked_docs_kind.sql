-- +goose Up
ALTER TABLE tracked_docs
    ADD COLUMN kind TEXT NOT NULL DEFAULT 'cv'
        CONSTRAINT tracked_docs_kind_check CHECK (kind IN ('cv', 'letter'));

-- +goose Down
ALTER TABLE tracked_docs DROP COLUMN kind;

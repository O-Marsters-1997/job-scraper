-- +goose Up
ALTER TABLE tracked_companies
    ADD COLUMN review_state TEXT NOT NULL DEFAULT 'kept' CHECK (review_state IN ('new', 'kept', 'dismissed'));

-- +goose Down
ALTER TABLE tracked_companies DROP COLUMN review_state;

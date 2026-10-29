-- +goose Up
ALTER TABLE scoring_options ADD COLUMN owner_user_id UUID REFERENCES users(id) ON DELETE CASCADE;
CREATE INDEX scoring_options_owner_live_idx ON scoring_options (owner_user_id) WHERE retired_at IS NULL;

-- +goose Down
DROP INDEX scoring_options_owner_live_idx;
ALTER TABLE scoring_options DROP COLUMN owner_user_id;

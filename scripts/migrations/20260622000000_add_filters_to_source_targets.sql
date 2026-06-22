-- +goose Up
ALTER TABLE source_targets ADD COLUMN filters JSONB NOT NULL DEFAULT '{}';
ALTER TABLE source_targets DROP CONSTRAINT source_targets_user_id_source_value_key;
ALTER TABLE source_targets ADD CONSTRAINT source_targets_user_id_source_value_filters_key
    UNIQUE (user_id, source, value, filters);

-- +goose Down
ALTER TABLE source_targets DROP CONSTRAINT source_targets_user_id_source_value_filters_key;
ALTER TABLE source_targets ADD CONSTRAINT source_targets_user_id_source_value_key
    UNIQUE (user_id, source, value);
ALTER TABLE source_targets DROP COLUMN filters;

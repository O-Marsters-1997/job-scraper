-- +goose Up
ALTER TABLE task_types
    ADD COLUMN system_key TEXT CHECK (system_key IN ('apply'));

UPDATE task_types SET system_key = 'apply' WHERE name = 'Apply';

CREATE UNIQUE INDEX task_types_user_system_key_idx
    ON task_types (user_id, system_key) WHERE system_key IS NOT NULL;

-- +goose Down
DROP INDEX task_types_user_system_key_idx;
ALTER TABLE task_types DROP COLUMN system_key;

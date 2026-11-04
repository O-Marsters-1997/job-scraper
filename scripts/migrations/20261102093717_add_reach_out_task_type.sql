-- +goose Up
ALTER TABLE task_types DROP CONSTRAINT task_types_system_key_check;
ALTER TABLE task_types
    ADD CONSTRAINT task_types_system_key_check CHECK (system_key IN ('apply', 'finish', 'reach_out'));

UPDATE task_types SET system_key = 'reach_out' WHERE name = 'Reach out';

INSERT INTO task_types (user_id, name, colour, system_key)
SELECT u.id, 'Reach out', '#15803d', 'reach_out'
FROM users u
WHERE NOT EXISTS (SELECT 1 FROM task_types tt WHERE tt.user_id = u.id AND tt.system_key = 'reach_out')
  AND NOT EXISTS (SELECT 1 FROM task_types tt WHERE tt.user_id = u.id AND tt.name = 'Reach out');

ALTER TABLE tasks ADD COLUMN contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL;

CREATE INDEX tasks_contact_open_idx ON tasks (contact_id) WHERE contact_id IS NOT NULL AND completed_at IS NULL;

-- +goose Down
DROP INDEX tasks_contact_open_idx;
ALTER TABLE tasks DROP COLUMN contact_id;
DELETE FROM tasks WHERE type_id IN (SELECT id FROM task_types WHERE system_key = 'reach_out');
DELETE FROM task_types WHERE system_key = 'reach_out';
ALTER TABLE task_types DROP CONSTRAINT task_types_system_key_check;
ALTER TABLE task_types
    ADD CONSTRAINT task_types_system_key_check CHECK (system_key IN ('apply', 'finish'));

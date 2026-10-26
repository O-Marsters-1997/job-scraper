-- +goose Up
ALTER TABLE task_types DROP CONSTRAINT task_types_system_key_check;
ALTER TABLE task_types
    ADD CONSTRAINT task_types_system_key_check CHECK (system_key IN ('apply', 'finish'));

UPDATE task_types SET system_key = 'finish' WHERE name = 'Finish application';

INSERT INTO task_types (user_id, name, colour, system_key)
SELECT u.id, 'Finish application', '#0f766e', 'finish'
FROM users u
WHERE NOT EXISTS (SELECT 1 FROM task_types tt WHERE tt.user_id = u.id AND tt.system_key = 'finish');

-- +goose Down
DELETE FROM tasks WHERE type_id IN (SELECT id FROM task_types WHERE system_key = 'finish');
DELETE FROM task_types WHERE system_key = 'finish';
ALTER TABLE task_types DROP CONSTRAINT task_types_system_key_check;
ALTER TABLE task_types
    ADD CONSTRAINT task_types_system_key_check CHECK (system_key IN ('apply'));

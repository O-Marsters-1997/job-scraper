-- +goose Up
ALTER TABLE application_statuses
    ADD COLUMN system_key TEXT CHECK (system_key IN ('draft', 'applied'));

UPDATE application_statuses s
SET system_key = lower(s.name)
FROM (
    SELECT DISTINCT ON (user_id, lower(name)) id
    FROM application_statuses
    WHERE lower(name) IN ('draft', 'applied')
    ORDER BY user_id, lower(name), created_at
) first
WHERE s.id = first.id;

INSERT INTO application_statuses (user_id, name, colour, system_key)
SELECT u.id, k.name, k.colour, k.key
FROM users u
CROSS JOIN (VALUES ('Draft', '#64748b', 'draft'), ('Applied', '#6366f1', 'applied')) AS k(name, colour, key)
WHERE NOT EXISTS (
    SELECT 1 FROM application_statuses s WHERE s.user_id = u.id AND s.system_key = k.key
);

CREATE UNIQUE INDEX application_statuses_user_system_key_idx
    ON application_statuses (user_id, system_key) WHERE system_key IS NOT NULL;

-- +goose Down
DROP INDEX application_statuses_user_system_key_idx;
ALTER TABLE application_statuses DROP COLUMN system_key;

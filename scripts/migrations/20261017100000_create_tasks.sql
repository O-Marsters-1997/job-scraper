-- +goose Up
CREATE TABLE task_types (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT        NOT NULL CHECK (length(name) BETWEEN 1 AND 50),
    colour     TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, name)
);

CREATE TABLE tasks (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type_id      UUID        NOT NULL REFERENCES task_types(id) ON DELETE CASCADE,
    job_id       UUID        REFERENCES jobs(id) ON DELETE SET NULL,
    link         TEXT        CHECK (link ~ '^https?://' AND length(link) <= 2048),
    notes        TEXT        CHECK (length(notes) <= 10000),
    completed_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX tasks_user_open_idx ON tasks (user_id, created_at DESC) WHERE completed_at IS NULL;
CREATE INDEX tasks_job_idx ON tasks (job_id) WHERE job_id IS NOT NULL;
CREATE INDEX tasks_type_idx ON tasks (type_id);

INSERT INTO task_types (user_id, name, colour)
SELECT u.id, d.name, d.colour
FROM users u
CROSS JOIN (VALUES
    ('Apply',     '#6366f1'),
    ('Follow up', '#c2410c'),
    ('Reach out', '#15803d'),
    ('Prep',      '#64748b')
) AS d(name, colour);

-- +goose Down
DROP TABLE tasks;
DROP TABLE task_types;

-- +goose Up
CREATE TABLE outreach_templates (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind       TEXT        NOT NULL
        CONSTRAINT outreach_templates_kind_check
        CHECK (kind IN ('recruiter', 'hiring_manager', 'referral', 'follow_up')),
    name       TEXT        NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
    body       TEXT        NOT NULL CHECK (length(body) BETWEEN 1 AND 2000),
    is_default BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, kind, name)
);
CREATE UNIQUE INDEX outreach_templates_default_idx
    ON outreach_templates (user_id, kind) WHERE is_default;

INSERT INTO outreach_templates (user_id, kind, name, body, is_default)
SELECT u.id, d.kind, d.name, d.body, TRUE
FROM users u
CROSS JOIN (VALUES
    ('recruiter', 'Agency intro', 'Hi {first_name}, I''m a backend engineer looking at {company}''s {role} roles. Recently I {achievement}. Open to a quick chat about what you''re hiring for?'),
    ('hiring_manager', 'Short and direct', 'Hi {first_name}, {hook} I {achievement}, and I''d love to bring that to {company}. Worth 15 minutes?'),
    ('referral', 'Referral ask', 'Hi {first_name}, I''m applying to {company} and saw you''re on the team. {hook} Would you be open to referring me, or telling me what the team''s like?'),
    ('follow_up', 'Gentle bump', 'Hi {first_name}, just bumping this in case it got buried. Still keen on {company}, happy to work around you.')
) AS d(kind, name, body);

-- +goose Down
DROP TABLE outreach_templates;

-- +goose Up
ALTER TABLE outreach_templates DROP CONSTRAINT outreach_templates_kind_check;
ALTER TABLE outreach_templates ADD CONSTRAINT outreach_templates_kind_check
    CHECK (kind IN ('just_applied', 'recruiter', 'hiring_manager', 'referral', 'follow_up'));
ALTER TABLE contact_events DROP CONSTRAINT contact_events_template_kind_check;
ALTER TABLE contact_events ADD CONSTRAINT contact_events_template_kind_check
    CHECK (template_kind IN ('just_applied', 'recruiter', 'hiring_manager', 'referral', 'follow_up'));

INSERT INTO outreach_templates (user_id, kind, name, body, is_default)
SELECT u.id, 'just_applied', 'Applied, CV attached',
    'Hi {first_name}, I''ve just applied for the {role} role at {company} and attached my CV. [hook] Happy to chat whenever suits.',
    TRUE
FROM users u;

-- +goose Down
UPDATE contact_events SET template_kind = NULL WHERE template_kind = 'just_applied';
DELETE FROM outreach_templates WHERE kind = 'just_applied';
ALTER TABLE contact_events DROP CONSTRAINT contact_events_template_kind_check;
ALTER TABLE contact_events ADD CONSTRAINT contact_events_template_kind_check
    CHECK (template_kind IN ('recruiter', 'hiring_manager', 'referral', 'follow_up'));
ALTER TABLE outreach_templates DROP CONSTRAINT outreach_templates_kind_check;
ALTER TABLE outreach_templates ADD CONSTRAINT outreach_templates_kind_check
    CHECK (kind IN ('recruiter', 'hiring_manager', 'referral', 'follow_up'));

-- +goose Up
INSERT INTO scoring_options (id, dimension, label, question) VALUES
    ('seniority:years_6plus', 'seniority', '6+ years', 'Does the role require 6 or more years of professional experience?'),
    ('seniority:people_lead', 'seniority', 'People lead', 'Is leading or line-managing other engineers a core part of the role?')
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM scoring_options WHERE id IN ('seniority:years_6plus', 'seniority:people_lead');

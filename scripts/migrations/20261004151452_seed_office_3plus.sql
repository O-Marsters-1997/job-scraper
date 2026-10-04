-- +goose Up
INSERT INTO scoring_options (id, dimension, label, question) VALUES
    ('work:office_3plus', 'work', 'Office 3+ days', 'Does the role expect you in an office 3 or more days a week?')
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM scoring_options WHERE id = 'work:office_3plus';

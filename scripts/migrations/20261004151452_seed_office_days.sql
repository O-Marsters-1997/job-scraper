-- +goose Up
INSERT INTO scoring_options (id, dimension, label, question) VALUES
    ('work:office_1', 'work', '1 office day/week', 'Does the role expect you in an office exactly 1 day a week? A range that includes 1, like "1-2 days", counts as yes.'),
    ('work:office_2', 'work', '2 office days/week', 'Does the role expect you in an office exactly 2 days a week? A range that includes 2, like "2-3 days", counts as yes.'),
    ('work:office_3', 'work', '3 office days/week', 'Does the role expect you in an office exactly 3 days a week? A range that includes 3, like "2-3 days", counts as yes.'),
    ('work:office_4plus', 'work', '4+ office days/week', 'Does the role expect you in an office 4 or more days a week? A range that reaches 4 or more, like "3-4 days", counts as yes.')
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM scoring_options WHERE id IN ('work:office_1', 'work:office_2', 'work:office_3', 'work:office_4plus');

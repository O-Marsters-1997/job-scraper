-- +goose Up
INSERT INTO scoring_options (id, dimension, label, question) VALUES
    ('size:startup', 'size', 'Startup', 'Does the company have under 50 employees (not a scale-up, large or enterprise company)?'),
    ('size:scaleup', 'size', 'Scale-up', 'Does the company have 50 to 500 employees (not under 50, or over 500)?'),
    ('size:large', 'size', 'Large', 'Does the company have 500 to 5,000 employees (not under 500, or over 5,000)?'),
    ('size:enterprise', 'size', 'Enterprise', 'Does the company have over 5,000 employees (not 5,000 or fewer)?')
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM scoring_options WHERE dimension = 'size';

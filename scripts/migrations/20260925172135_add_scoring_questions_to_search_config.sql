-- +goose Up
ALTER TABLE search_config ADD COLUMN scoring_questions JSONB NOT NULL DEFAULT '{}';

UPDATE search_config
SET scoring_questions = jsonb_build_object(
    'profile', suitability_rubric,
    'criteria', '[]'::jsonb,
    'scale', jsonb_build_array('Not relevant', 'Weak', 'Possible', 'Strong', 'Apply today')
);

-- +goose Down
ALTER TABLE search_config DROP COLUMN scoring_questions;

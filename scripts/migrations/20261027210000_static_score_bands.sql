-- +goose Up
UPDATE search_config
SET preferences = preferences #- '{scoring,bands}'
WHERE preferences->'scoring' ? 'bands';

UPDATE job_scores
SET band = CASE
        WHEN suitability_score >= 80 THEN 'great'
        WHEN suitability_score >= 65 THEN 'good'
        WHEN suitability_score >= 45 THEN 'fair'
        ELSE 'poor'
    END
WHERE suitability_score IS NOT NULL;

-- +goose Down
SELECT 1;

-- +goose NO TRANSACTION

-- +goose Up
CREATE INDEX CONCURRENTLY job_scores_user_best_idx ON job_scores (user_id, suitability_score DESC NULLS LAST) WHERE NOT blocked;

-- +goose Down
DROP INDEX CONCURRENTLY job_scores_user_best_idx;

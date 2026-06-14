-- name: UpsertJobScoreRelevance :exec
INSERT INTO job_scores (job_id, user_id, relevance_score)
VALUES ($1, $2, $3)
ON CONFLICT (job_id, user_id) DO UPDATE SET
    relevance_score = EXCLUDED.relevance_score,
    updated_at = NOW();

-- name: UpsertJobScoreSuitability :exec
INSERT INTO job_scores (job_id, user_id, suitability_score)
VALUES ($1, $2, $3)
ON CONFLICT (job_id, user_id) DO UPDATE SET
    suitability_score = EXCLUDED.suitability_score,
    updated_at = NOW();

-- name: GetJobScore :one
SELECT * FROM job_scores WHERE job_id = $1 AND user_id = $2 LIMIT 1;

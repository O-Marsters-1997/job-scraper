-- name: GetJobForScoring :one
SELECT id, title, location, url, company_slug, source, updated_at, scraped_at, description,
    salary_raw, work_arrangement, company_id, primary_board_id, provider_posting_id, content_fingerprint
FROM jobs WHERE id = $1;

-- name: ListInterestedConfigs :many
SELECT u.id AS user_id,
    COALESCE(sc.excluded_title_keywords, '{}')::text[] AS excluded_title_keywords,
    COALESCE(sc.excluded_companies, '{}')::text[] AS excluded_companies,
    COALESCE(sc.excluded_locations, '{}')::text[] AS excluded_locations,
    COALESCE(sc.required_locations, '{}')::text[] AS required_locations,
    COALESCE(sc.required_title_keywords, '{}')::text[] AS required_title_keywords,
    COALESCE(sc.notify_threshold, 70) AS notify_threshold,
    COALESCE(sc.preferences, '{}'::jsonb) AS preferences,
    EXISTS (
        SELECT 1 FROM tracked_companies tc JOIN companies c ON c.id = tc.company_id
        WHERE tc.user_id = u.id AND tc.enabled AND tc.review_state = 'new'
          AND (c.id = j.company_id OR c.slug = j.company_slug)
    ) AS company_is_new
FROM users u
LEFT JOIN search_config sc ON sc.user_id = u.id
JOIN jobs j ON j.id = sqlc.arg(job_id)::uuid
WHERE EXISTS (
    SELECT 1 FROM tracked_companies tc JOIN companies c ON c.id = tc.company_id
    WHERE tc.user_id = u.id AND tc.enabled AND (c.id = j.company_id OR c.slug = j.company_slug)
) OR EXISTS (
    SELECT 1 FROM source_targets st WHERE st.user_id = u.id AND st.enabled AND st.source = j.source
);

-- name: ListOptionAnswers :many
SELECT question_hash, p_yes, p_no, p_not_stated, confidence
FROM option_answers
WHERE job_id = sqlc.arg(job_id)::uuid AND fingerprint = sqlc.arg(fingerprint)::text AND model = sqlc.arg(model)::text;

-- name: InsertOptionAnswer :exec
INSERT INTO option_answers (job_id, fingerprint, question_hash, model, p_yes, p_no, p_not_stated, confidence)
VALUES (sqlc.arg(job_id)::uuid, sqlc.arg(fingerprint)::text, sqlc.arg(question_hash)::text, sqlc.arg(model)::text,
    sqlc.arg(p_yes)::real, sqlc.arg(p_no)::real, sqlc.arg(p_not_stated)::real, sqlc.arg(confidence)::real)
ON CONFLICT (job_id, fingerprint, question_hash, model) DO NOTHING;

-- name: DropStaleAnswers :exec
DELETE FROM option_answers
USING jobs j
WHERE option_answers.job_id = j.id
    AND option_answers.job_id = ANY(sqlc.arg(job_ids)::uuid[])
    AND option_answers.fingerprint != j.content_fingerprint;

-- name: DeleteAnswersForJobs :exec
DELETE FROM option_answers WHERE job_id = ANY(sqlc.arg(job_ids)::uuid[]);

-- name: UpsertJobScore :exec
INSERT INTO job_scores (job_id, user_id, suitability_score, band, breakdown, cost, score_fingerprint, score_model)
VALUES (sqlc.arg(job_id)::uuid, sqlc.arg(user_id)::uuid, sqlc.arg(score)::int, sqlc.arg(band)::text, sqlc.arg(breakdown)::jsonb,
    sqlc.narg(cost)::numeric, sqlc.arg(fingerprint)::text, sqlc.arg(model)::text)
ON CONFLICT (job_id, user_id) DO UPDATE SET
    suitability_score = EXCLUDED.suitability_score, band = EXCLUDED.band, breakdown = EXCLUDED.breakdown, cost = EXCLUDED.cost,
    score_fingerprint = EXCLUDED.score_fingerprint, score_model = EXCLUDED.score_model, updated_at = NOW();

-- name: UpdateJobScoreBreakdown :exec
UPDATE job_scores SET suitability_score = sqlc.arg(score)::int, band = sqlc.arg(band)::text, breakdown = sqlc.arg(breakdown)::jsonb, updated_at = NOW()
WHERE job_id = sqlc.arg(job_id)::uuid AND user_id = sqlc.arg(user_id)::uuid;

-- name: ListScoringInputJobs :many
SELECT j.id, j.title, j.location, j.url, j.company_slug, j.source, j.updated_at, j.scraped_at,
    j.description, j.salary_raw, j.work_arrangement, j.company_id, j.primary_board_id,
    j.provider_posting_id, j.content_fingerprint
FROM job_scores s JOIN jobs j ON j.id = s.job_id
WHERE s.user_id = sqlc.arg(user_id)::uuid;

-- name: ListScoringAnswersForUser :many
SELECT j.id AS job_id, a.question_hash, a.p_yes, a.p_no, a.p_not_stated, a.confidence
FROM job_scores s
JOIN jobs j ON j.id = s.job_id
JOIN option_answers a ON a.job_id = j.id AND a.fingerprint = j.content_fingerprint
WHERE s.user_id = sqlc.arg(user_id)::uuid AND a.model = sqlc.arg(model)::text;

-- name: GetScoringStatus :one
SELECT count(*) AS pending
FROM effect_outbox e
JOIN job_scores s ON s.job_id = e.job_id AND s.user_id = sqlc.arg(user_id)::uuid
WHERE e.status IN ('pending', 'running');

-- name: ListOpenCompanyJobs :many
SELECT id, company_id FROM jobs
WHERE company_id = ANY(sqlc.arg(company_ids)::uuid[]) AND closed_at IS NULL;

-- name: ListCompanyJobAnswers :many
SELECT j.id AS job_id, a.question_hash, a.p_yes, a.p_no, a.p_not_stated, a.confidence
FROM jobs j
JOIN option_answers a ON a.job_id = j.id AND a.fingerprint = j.content_fingerprint
WHERE j.company_id = ANY(sqlc.arg(company_ids)::uuid[]) AND j.closed_at IS NULL AND a.model = sqlc.arg(model)::text;

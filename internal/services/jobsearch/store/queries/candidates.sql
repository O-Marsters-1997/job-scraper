-- name: UpsertCandidate :one
INSERT INTO job_candidates (normalized_url, source, card_title, card_company, card_location, card, posted_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (normalized_url) DO UPDATE SET
    source = EXCLUDED.source,
    card_title = EXCLUDED.card_title,
    card_company = EXCLUDED.card_company,
    card_location = EXCLUDED.card_location,
    card = EXCLUDED.card,
    posted_at = LEAST(job_candidates.posted_at, EXCLUDED.posted_at),
    last_seen_at = NOW(),
    expires_at = NOW() + INTERVAL '60 days'
RETURNING id;

-- name: RecordCandidateDiscovery :exec
INSERT INTO candidate_discoveries (candidate_id, source_target_id)
VALUES ($1, $2)
ON CONFLICT (candidate_id, source_target_id) DO UPDATE SET last_seen_at = NOW();

-- name: ListCandidatesForUser :many
SELECT c.id, c.normalized_url, c.card_title, c.card_company, c.card_location, c.card, c.posted_at, c.source
FROM job_candidates c
WHERE c.id > $2 AND c.expires_at > NOW()
  AND EXISTS (
      SELECT 1 FROM candidate_discoveries d
      JOIN source_targets t ON t.id = d.source_target_id
      WHERE d.candidate_id = c.id AND t.user_id = $1 AND t.enabled
  )
ORDER BY c.id
LIMIT $3;

-- name: AssessCandidate :one
WITH assessment AS (
    INSERT INTO candidate_assessments (candidate_id, user_id, search_config_version, relevance)
    VALUES ($1, $2, $3, $4)
    ON CONFLICT (candidate_id, user_id) DO UPDATE SET
        search_config_version = EXCLUDED.search_config_version,
        relevance = EXCLUDED.relevance,
        evaluated_at = NOW()
    RETURNING candidate_id
), claimed AS (
    SELECT c.id FROM job_candidates c
    WHERE c.id = (SELECT candidate_id FROM assessment)
      AND $4::boolean AND c.detail_state = 'unrequested' AND c.expires_at > NOW()
      AND NOT EXISTS (SELECT 1 FROM jobs j WHERE j.url = c.normalized_url)
)
SELECT EXISTS (SELECT 1 FROM claimed);

-- name: MarkCandidateDetailPending :exec
UPDATE job_candidates SET detail_state = 'pending' WHERE id = $1;

-- name: UpsertBoardCardDate :exec
INSERT INTO job_candidates (normalized_url, source, card_title, card_company, card_location, posted_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (normalized_url) DO UPDATE SET
    posted_at = LEAST(job_candidates.posted_at, EXCLUDED.posted_at);

-- name: ListDatedCandidatesForCompany :many
SELECT card_title, card_location, posted_at
FROM job_candidates
WHERE card_company = $1 AND posted_at IS NOT NULL AND expires_at > NOW()
ORDER BY posted_at
LIMIT 200;

-- name: DeleteExpiredCandidates :exec
DELETE FROM job_candidates WHERE expires_at <= NOW();

-- name: ListCandidateDiscoveryFilters :many
SELECT d.candidate_id, t.filters
FROM candidate_discoveries d
JOIN source_targets t ON t.id = d.source_target_id
WHERE d.candidate_id = ANY(sqlc.arg(candidate_ids)::uuid[]) AND t.user_id = sqlc.arg(user_id) AND t.enabled;

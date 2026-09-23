-- +goose Up
CREATE TABLE tracked_companies (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    check_interval_minutes INT NOT NULL DEFAULT 360 CHECK (check_interval_minutes >= 60),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, company_id)
);

CREATE INDEX tracked_companies_company_enabled_idx ON tracked_companies (company_id, check_interval_minutes) WHERE enabled;

CREATE TABLE tracking_backfill_issues (
    source_target_id UUID PRIMARY KEY,
    reason TEXT NOT NULL,
    candidate_company_ids UUID[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TEMP TABLE tracking_backfill_candidates ON COMMIT DROP AS
SELECT st.id AS source_target_id, st.user_id, st.enabled, st.check_interval_minutes,
       c.id AS company_id,
       COUNT(c.id) OVER (PARTITION BY st.id) AS match_count
FROM source_targets st
LEFT JOIN companies c
  ON c.id = st.company_id
  OR (st.company_id IS NULL AND c.ats_source = st.source AND c.ats_token = st.value)
WHERE st.source IN ('greenhouse', 'lever', 'ashby', 'workable', 'recruitee', 'personio');

INSERT INTO tracking_backfill_issues (source_target_id, reason, candidate_company_ids)
SELECT source_target_id,
       CASE WHEN MAX(match_count) = 0 THEN 'unmatched' ELSE 'ambiguous_company_match' END,
       COALESCE(ARRAY_AGG(company_id) FILTER (WHERE company_id IS NOT NULL), '{}')
FROM tracking_backfill_candidates
WHERE match_count <> 1
GROUP BY source_target_id;

INSERT INTO tracking_backfill_issues (source_target_id, reason, candidate_company_ids)
SELECT source_target_id, 'frequency_collision', ARRAY[company_id]
FROM tracking_backfill_candidates bc
WHERE match_count = 1
  AND EXISTS (
      SELECT 1 FROM tracking_backfill_candidates other
      WHERE other.user_id = bc.user_id AND other.company_id = bc.company_id
        AND other.match_count = 1 AND other.check_interval_minutes <> bc.check_interval_minutes
  )
ON CONFLICT (source_target_id) DO NOTHING;

INSERT INTO tracked_companies (user_id, company_id, enabled, check_interval_minutes)
SELECT user_id, company_id, BOOL_OR(enabled), MIN(check_interval_minutes)
FROM tracking_backfill_candidates
WHERE match_count = 1
GROUP BY user_id, company_id;

-- +goose Down
DROP TABLE tracking_backfill_issues;
DROP TABLE tracked_companies;

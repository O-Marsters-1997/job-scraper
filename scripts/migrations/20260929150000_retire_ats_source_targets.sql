-- +goose Up
INSERT INTO tracked_companies (user_id, company_id, enabled, check_interval_minutes)
SELECT DISTINCT ON (st.user_id, c.id) st.user_id, c.id, st.enabled, st.check_interval_minutes
FROM source_targets st
JOIN companies c
  ON c.id = st.company_id
  OR (st.company_id IS NULL AND c.ats_source = st.source AND c.ats_token = st.value)
WHERE st.source IN ('greenhouse', 'lever', 'ashby', 'workable', 'recruitee', 'personio')
ORDER BY st.user_id, c.id, st.enabled DESC, st.updated_at DESC
ON CONFLICT (user_id, company_id) DO NOTHING;

DELETE FROM source_targets
WHERE source IN ('greenhouse', 'lever', 'ashby', 'workable', 'recruitee', 'personio');

-- +goose Down
SELECT 1;

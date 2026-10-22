-- name: PageJobs :many
SELECT j.id, j.title, j.location, j.url, j.company_slug, j.source, j.posted_at, j.posted_after, j.reposted_at, j.scraped_at, j.first_discovered_at, j.salary_raw, j.work_arrangement, j.company_id, j.primary_board_id, j.provider_posting_id, j.content_fingerprint, js.suitability_score, js.band, js.breakdown, COALESCE(jg.grade, '')::text AS grade, (jv.job_id IS NOT NULL)::bool AS seen, EXISTS (SELECT 1 FROM company_favourites cf WHERE cf.user_id = sqlc.arg(user_id)::uuid AND (cf.company_id = j.company_id OR cf.company_id = (SELECT fc.id FROM companies fc WHERE fc.slug = j.company_slug))) AS company_favourite, COALESCE((SELECT kc.kind FROM companies kc WHERE kc.id = j.company_id), (SELECT kc.kind FROM companies kc WHERE kc.slug = j.company_slug), 'unclassified')::text AS company_kind, COALESCE(j.discovered_via, (SELECT cb.discovered_via FROM company_boards cb WHERE cb.id = j.primary_board_id), '')::text AS discovered_via
FROM jobs j
LEFT JOIN job_scores js ON js.job_id = j.id AND js.user_id = sqlc.arg(user_id)::uuid
LEFT JOIN job_grades jg ON jg.job_id = j.id AND jg.user_id = sqlc.arg(user_id)::uuid
LEFT JOIN job_views jv ON jv.job_id = j.id AND jv.user_id = sqlc.arg(user_id)::uuid
WHERE j.closed_at IS NULL
  AND COALESCE(j.posted_at, j.posted_after, j.first_discovered_at) > sqlc.arg(now)::timestamptz - interval '90 days'
  AND COALESCE(j.company_slug, '') <> ALL(sqlc.arg(excluded_company_slugs)::text[])
  AND NOT COALESCE(js.breakdown @> '[{"effect":"blocked"}]'::jsonb, false)
  AND jg.grade IS DISTINCT FROM 'no'
  AND (sqlc.narg(company_id)::uuid IS NULL OR j.company_id = sqlc.narg(company_id)::uuid OR (j.company_id IS NULL AND j.company_slug = (SELECT slug FROM companies WHERE id = sqlc.narg(company_id)::uuid)))
  AND (NOT sqlc.arg(scored_only)::bool OR js.job_id IS NOT NULL)
  AND (sqlc.narg(min_suitability)::int IS NULL OR js.suitability_score >= sqlc.narg(min_suitability)::int)
  AND (cardinality(sqlc.arg(bands)::text[]) = 0 OR js.band = ANY(sqlc.arg(bands)::text[]))
  AND (sqlc.narg(scraped_since)::timestamptz IS NULL OR j.scraped_at >= sqlc.narg(scraped_since)::timestamptz)
  AND (sqlc.arg(graded)::text = 'all' OR (sqlc.arg(graded)::text = 'graded') = (COALESCE(jg.grade, '') <> ''))
  AND (sqlc.arg(seen)::text = 'all' OR (sqlc.arg(seen)::text = 'seen') = (jv.job_id IS NOT NULL))
  AND (sqlc.arg(poster)::text = 'all' OR (sqlc.arg(poster)::text = 'agency') = (COALESCE((SELECT kc.kind FROM companies kc WHERE kc.id = j.company_id), (SELECT kc.kind FROM companies kc WHERE kc.slug = j.company_slug), 'unclassified') IN ('agency', 'likely_agency')))
  AND (NOT sqlc.arg(favourite_only)::bool OR EXISTS (SELECT 1 FROM company_favourites cf WHERE cf.user_id = sqlc.arg(user_id)::uuid AND (cf.company_id = j.company_id OR cf.company_id = (SELECT fc.id FROM companies fc WHERE fc.slug = j.company_slug))))
  AND (cardinality(sqlc.arg(sources)::text[]) = 0 OR j.source = ANY(sqlc.arg(sources)::text[]))
  AND (cardinality(sqlc.arg(arrangements)::text[]) = 0 OR (CASE lower(btrim(j.work_arrangement)) WHEN 'remote' THEN 'remote' WHEN 'hybrid' THEN 'hybrid' WHEN 'onsite' THEN 'onsite' ELSE 'unknown' END) = ANY(sqlc.arg(arrangements)::text[]))
  AND (NOT sqlc.arg(salary_only)::bool OR (
    j.salary_min IS NOT NULL
    AND COALESCE(j.salary_max, j.salary_min) >= COALESCE(sqlc.narg(salary_floor)::float8, '-Infinity'::float8)
    AND j.salary_min <= COALESCE(sqlc.narg(salary_ceiling)::float8, 'Infinity'::float8)))
  AND (sqlc.arg(search_pattern)::text = '' OR concat_ws(' ', j.title, j.company_slug, j.location, j.source) ILIKE sqlc.arg(search_pattern)::text)
ORDER BY
  CASE WHEN sqlc.arg(sort)::text NOT IN ('newest', 'best') THEN
    COALESCE(js.suitability_score, 50)::float8 * power(0.5::float8, GREATEST(0, extract(epoch FROM sqlc.arg(now)::timestamptz - COALESCE(j.posted_at, j.first_discovered_at))) / 86400 / 3)
  END DESC NULLS LAST,
  CASE WHEN sqlc.arg(sort)::text = 'best' THEN js.suitability_score END DESC NULLS LAST,
  CASE WHEN sqlc.arg(sort)::text NOT IN ('newest', 'best') THEN j.first_discovered_at END DESC,
  CASE WHEN sqlc.arg(sort)::text IN ('newest', 'best') THEN COALESCE(j.posted_at, j.first_discovered_at) END DESC,
  j.id
LIMIT sqlc.arg(page_limit)::int OFFSET sqlc.arg(page_offset)::int;

-- name: CountJobs :one
SELECT count(*)
FROM jobs j
LEFT JOIN job_scores js ON js.job_id = j.id AND js.user_id = sqlc.arg(user_id)::uuid
LEFT JOIN job_grades jg ON jg.job_id = j.id AND jg.user_id = sqlc.arg(user_id)::uuid
LEFT JOIN job_views jv ON jv.job_id = j.id AND jv.user_id = sqlc.arg(user_id)::uuid
WHERE j.closed_at IS NULL
  AND COALESCE(j.posted_at, j.posted_after, j.first_discovered_at) > sqlc.arg(now)::timestamptz - interval '90 days'
  AND COALESCE(j.company_slug, '') <> ALL(sqlc.arg(excluded_company_slugs)::text[])
  AND NOT COALESCE(js.breakdown @> '[{"effect":"blocked"}]'::jsonb, false)
  AND jg.grade IS DISTINCT FROM 'no'
  AND (sqlc.narg(company_id)::uuid IS NULL OR j.company_id = sqlc.narg(company_id)::uuid OR (j.company_id IS NULL AND j.company_slug = (SELECT slug FROM companies WHERE id = sqlc.narg(company_id)::uuid)))
  AND (NOT sqlc.arg(scored_only)::bool OR js.job_id IS NOT NULL)
  AND (sqlc.narg(min_suitability)::int IS NULL OR js.suitability_score >= sqlc.narg(min_suitability)::int)
  AND (cardinality(sqlc.arg(bands)::text[]) = 0 OR js.band = ANY(sqlc.arg(bands)::text[]))
  AND (sqlc.narg(scraped_since)::timestamptz IS NULL OR j.scraped_at >= sqlc.narg(scraped_since)::timestamptz)
  AND (sqlc.arg(graded)::text = 'all' OR (sqlc.arg(graded)::text = 'graded') = (COALESCE(jg.grade, '') <> ''))
  AND (sqlc.arg(seen)::text = 'all' OR (sqlc.arg(seen)::text = 'seen') = (jv.job_id IS NOT NULL))
  AND (sqlc.arg(poster)::text = 'all' OR (sqlc.arg(poster)::text = 'agency') = (COALESCE((SELECT kc.kind FROM companies kc WHERE kc.id = j.company_id), (SELECT kc.kind FROM companies kc WHERE kc.slug = j.company_slug), 'unclassified') IN ('agency', 'likely_agency')))
  AND (NOT sqlc.arg(favourite_only)::bool OR EXISTS (SELECT 1 FROM company_favourites cf WHERE cf.user_id = sqlc.arg(user_id)::uuid AND (cf.company_id = j.company_id OR cf.company_id = (SELECT fc.id FROM companies fc WHERE fc.slug = j.company_slug))))
  AND (cardinality(sqlc.arg(sources)::text[]) = 0 OR j.source = ANY(sqlc.arg(sources)::text[]))
  AND (cardinality(sqlc.arg(arrangements)::text[]) = 0 OR (CASE lower(btrim(j.work_arrangement)) WHEN 'remote' THEN 'remote' WHEN 'hybrid' THEN 'hybrid' WHEN 'onsite' THEN 'onsite' ELSE 'unknown' END) = ANY(sqlc.arg(arrangements)::text[]))
  AND (NOT sqlc.arg(salary_only)::bool OR (
    j.salary_min IS NOT NULL
    AND COALESCE(j.salary_max, j.salary_min) >= COALESCE(sqlc.narg(salary_floor)::float8, '-Infinity'::float8)
    AND j.salary_min <= COALESCE(sqlc.narg(salary_ceiling)::float8, 'Infinity'::float8)))
  AND (sqlc.arg(search_pattern)::text = '' OR concat_ws(' ', j.title, j.company_slug, j.location, j.source) ILIKE sqlc.arg(search_pattern)::text);

-- name: GetJob :one
SELECT j.id, j.title, j.location, j.url, j.company_slug, j.source, j.posted_at, j.posted_after, j.reposted_at, j.scraped_at, j.first_discovered_at, j.description, j.salary_raw, j.work_arrangement, j.company_id, j.primary_board_id, j.provider_posting_id, j.content_fingerprint, js.suitability_score, js.band, js.breakdown, (jv.job_id IS NOT NULL)::bool AS seen, EXISTS (SELECT 1 FROM company_favourites cf JOIN companies fc ON fc.id = cf.company_id WHERE cf.user_id = $2 AND (fc.id = j.company_id OR fc.slug = j.company_slug)) AS company_favourite, COALESCE((SELECT kc.kind FROM companies kc WHERE kc.id = j.company_id OR kc.slug = j.company_slug ORDER BY (kc.id = j.company_id) DESC LIMIT 1), 'unclassified')::text AS company_kind, COALESCE(j.discovered_via, (SELECT cb.discovered_via FROM company_boards cb WHERE cb.id = j.primary_board_id), '')::text AS discovered_via
FROM jobs j
LEFT JOIN job_scores js ON js.job_id = j.id AND js.user_id = $2
LEFT JOIN job_views jv ON jv.job_id = j.id AND jv.user_id = $2
WHERE j.id = $1
LIMIT 1;

-- name: ExistingURLs :many
SELECT url FROM jobs WHERE url = ANY($1::text[])
UNION
SELECT normalized_url FROM job_candidates WHERE normalized_url = ANY($1::text[]);

-- name: MarkJobsSeen :exec
INSERT INTO job_views (user_id, job_id)
SELECT sqlc.arg(user_id)::uuid, j.id
FROM jobs j
WHERE j.id = ANY(sqlc.arg(job_ids)::uuid[])
ON CONFLICT DO NOTHING;

-- name: MarkJobsUnseen :exec
DELETE FROM job_views
WHERE user_id = sqlc.arg(user_id)::uuid AND job_id = ANY(sqlc.arg(job_ids)::uuid[]);

-- name: ListJobListings :many
SELECT source, normalized_url, first_seen_at
FROM job_urls
WHERE job_id = $1
ORDER BY first_seen_at, normalized_url;

-- name: ListJobListingsForJobs :many
SELECT job_id, source, normalized_url, first_seen_at
FROM job_urls
WHERE job_id = ANY($1::uuid[])
ORDER BY first_seen_at, normalized_url;

-- name: JobStats :one
SELECT count(*)::int AS total,
       count(DISTINCT j.source)::int AS sources,
       (count(*) FILTER (WHERE j.scraped_at >= sqlc.arg(day_start)::timestamptz))::int AS new_today,
       (count(DISTINCT j.source) FILTER (WHERE j.scraped_at >= sqlc.arg(day_start)::timestamptz))::int AS today_sources
FROM jobs j
JOIN job_scores js ON js.job_id = j.id AND js.user_id = sqlc.arg(user_id)::uuid
LEFT JOIN job_grades jg ON jg.job_id = j.id AND jg.user_id = sqlc.arg(user_id)::uuid
WHERE j.closed_at IS NULL
  AND COALESCE(j.posted_at, j.posted_after, j.first_discovered_at) > sqlc.arg(now)::timestamptz - interval '90 days'
  AND COALESCE(j.company_slug, '') <> ALL(sqlc.arg(excluded_company_slugs)::text[])
  AND NOT COALESCE(js.breakdown @> '[{"effect":"blocked"}]'::jsonb, false)
  AND jg.grade IS DISTINCT FROM 'no';

\set ON_ERROR_STOP on
\pset null '-'

CREATE TEMP VIEW weeks AS
SELECT w AS week_start, w + interval '7 days' AS week_end
FROM generate_series(
    date_trunc('week', now() AT TIME ZONE 'UTC') - interval '28 days',
    date_trunc('week', now() AT TIME ZONE 'UTC') - interval '7 days',
    interval '7 days') AS w;

CREATE TEMP VIEW scored_jobs AS
SELECT j.id, j.source, j.primary_board_id, c.kind AS company_kind,
       (j.first_discovered_at AT TIME ZONE 'UTC') AS discovered_at
FROM jobs j
LEFT JOIN companies c ON c.id = j.company_id
WHERE EXISTS (
    SELECT 1 FROM job_scores s
    WHERE s.job_id = j.id AND s.band IN ('great', 'good'));

\echo '== Headline: relevant direct Jobs per week, by Board discovered_via =='
SELECT w.week_start::date AS week,
       COALESCE(b.discovered_via, 'user-added') AS discovered_via,
       count(*) AS relevant_direct_jobs
FROM weeks w
JOIN scored_jobs sj ON sj.discovered_at >= w.week_start AND sj.discovered_at < w.week_end
JOIN company_boards b ON b.id = sj.primary_board_id
WHERE sj.company_kind IS DISTINCT FROM 'agency'
  AND sj.company_kind IS DISTINCT FROM 'likely_agency'
GROUP BY 1, 2
ORDER BY 1, 2;

\echo '== Shares: LinkedIn and agency share of great/good Jobs per week =='
SELECT w.week_start::date AS week,
       count(sj.id) AS great_good_jobs,
       count(sj.id) FILTER (WHERE sj.source = 'linkedin') AS linkedin_jobs,
       round(100.0 * count(sj.id) FILTER (WHERE sj.source = 'linkedin') / NULLIF(count(sj.id), 0), 1) AS linkedin_pct,
       count(sj.id) FILTER (WHERE sj.company_kind IN ('agency', 'likely_agency')) AS agency_jobs,
       round(100.0 * count(sj.id) FILTER (WHERE sj.company_kind IN ('agency', 'likely_agency')) / NULLIF(count(sj.id), 0), 1) AS agency_pct
FROM weeks w
LEFT JOIN scored_jobs sj ON sj.discovered_at >= w.week_start AND sj.discovered_at < w.week_end
GROUP BY 1
ORDER BY 1;

\echo '== Boards: verified Boards by discovered_via =='
SELECT COALESCE(discovered_via, 'user-added') AS discovered_via, count(*) AS verified_boards
FROM company_boards
WHERE status = 'verified'
GROUP BY 1
ORDER BY 1;

\echo '== Boards: tracked companies by discovered_via of their verified Boards =='
SELECT COALESCE(b.discovered_via, 'user-added') AS discovered_via,
       count(DISTINCT tc.company_id) AS tracked_companies
FROM tracked_companies tc
JOIN company_boards b ON b.company_id = tc.company_id AND b.status = 'verified'
GROUP BY 1
ORDER BY 1;

\echo '== Export verified Boards for the hiring.cafe coverage comm =='
\echo 'Run: psql -At -c "SELECT lower(source || '':'' || board_token) FROM company_boards WHERE status = ''verified''" "$PROD_URL" | LC_ALL=C sort -u > boards.txt'
\echo 'Then: comm -12 .claude/handoffs/sme-discovery-data/hiringcafe-london-supported-boards.txt boards.txt | wc -l'

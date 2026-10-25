package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const wipeAllButRealUserRowsSQL = `
DELETE FROM users WHERE username LIKE 'perf-%';
DELETE FROM job_scores;
DELETE FROM job_rejections;
DELETE FROM scrape_runs;

CREATE TEMP TABLE keep_jobs (id uuid PRIMARY KEY) ON COMMIT DROP;
CREATE TEMP TABLE keep_companies (id uuid PRIMARY KEY) ON COMMIT DROP;
CREATE TEMP TABLE keep_boards (id uuid PRIMARY KEY) ON COMMIT DROP;

CREATE TEMP VIEW fk_refs AS
SELECT c.conrelid::regclass AS tbl, a.attname AS col, c.confrelid::regclass::text AS ref,
    EXISTS (SELECT 1 FROM pg_attribute u WHERE u.attrelid = c.conrelid AND u.attname = 'user_id' AND NOT u.attisdropped) AS user_owned
FROM pg_constraint c
JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
WHERE c.contype = 'f' AND c.confrelid::regclass::text IN ('jobs', 'companies', 'company_boards');

DO $$
DECLARE r record;
BEGIN
  FOR r IN SELECT * FROM fk_refs WHERE user_owned AND ref IN ('jobs', 'companies') LOOP
    EXECUTE format('INSERT INTO %s SELECT DISTINCT %I FROM %s WHERE %I IS NOT NULL ON CONFLICT DO NOTHING',
      CASE WHEN r.ref = 'jobs' THEN 'keep_jobs' ELSE 'keep_companies' END, r.col, r.tbl, r.col);
  END LOOP;
END $$;

INSERT INTO keep_companies SELECT DISTINCT company_id FROM jobs WHERE id IN (SELECT id FROM keep_jobs) AND company_id IS NOT NULL
ON CONFLICT DO NOTHING;
INSERT INTO keep_boards SELECT id FROM company_boards WHERE company_id IN (SELECT id FROM keep_companies);

SET LOCAL session_replication_role = replica;

DO $$
DECLARE r record;
BEGIN
  FOR r IN SELECT * FROM fk_refs LOOP
    EXECUTE format('DELETE FROM %s WHERE %I IS NOT NULL AND %I NOT IN (SELECT id FROM %s)', r.tbl, r.col, r.col,
      CASE r.ref WHEN 'jobs' THEN 'keep_jobs' WHEN 'companies' THEN 'keep_companies' ELSE 'keep_boards' END);
  END LOOP;
END $$;

DELETE FROM jobs WHERE id NOT IN (SELECT id FROM keep_jobs);
DELETE FROM company_boards WHERE id NOT IN (SELECT id FROM keep_boards);
DELETE FROM companies WHERE id NOT IN (SELECT id FROM keep_companies);

SET LOCAL session_replication_role = origin;
DROP VIEW fk_refs;
`

func wipe(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, wipeAllButRealUserRowsSQL); err != nil {
		return fmt.Errorf("wipe dev data: %w", err)
	}
	return nil
}

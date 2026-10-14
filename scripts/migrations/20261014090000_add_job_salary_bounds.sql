-- +goose Up
ALTER TABLE jobs ADD COLUMN salary_min DOUBLE PRECISION, ADD COLUMN salary_max DOUBLE PRECISION;

UPDATE jobs j SET salary_min = b.lo, salary_max = COALESCE(b.hi, b.lo)
FROM (
    SELECT j2.id,
        (array_agg(n.v ORDER BY n.ord))[1] AS lo,
        (array_agg(n.v ORDER BY n.ord))[2] AS hi
    FROM jobs j2
    CROSS JOIN LATERAL (
        SELECT COALESCE(o.m[1]::double precision * 1000, o.m[2]::double precision) AS v, o.ord
        FROM regexp_matches(
            regexp_replace(j2.salary_raw, '[£$€,]', '', 'g'),
            '\y(\d{1,15}(?:\.\d+)?)k\y|(\d{1,15}(?:\.\d+)?)', 'gi'
        ) WITH ORDINALITY AS o(m, ord)
    ) n
    WHERE j2.salary_raw <> ''
    GROUP BY j2.id
) b
WHERE j.id = b.id;

-- +goose Down
ALTER TABLE jobs DROP COLUMN salary_max, DROP COLUMN salary_min;

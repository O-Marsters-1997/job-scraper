-- +goose Up
CREATE TABLE company_boards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    source TEXT NOT NULL,
    board_token TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'candidate' CHECK (status IN ('candidate', 'verified', 'retired')),
    verification_method TEXT,
    verified_at TIMESTAMPTZ,
    last_linked_at TIMESTAMPTZ,
    retired_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source, board_token),
    CHECK (status <> 'verified' OR (verification_method IS NOT NULL AND verified_at IS NOT NULL))
);
CREATE INDEX company_boards_company_status_idx ON company_boards (company_id, status);

CREATE TABLE board_backfill_issues (
    source TEXT NOT NULL,
    board_token TEXT NOT NULL,
    candidate_company_ids UUID[] NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (source, board_token)
);

INSERT INTO board_backfill_issues (source, board_token, candidate_company_ids)
SELECT ats_source, ats_token, ARRAY_AGG(id ORDER BY first_seen_at, id)
FROM companies
WHERE ats_source IS NOT NULL AND ats_token IS NOT NULL
GROUP BY ats_source, ats_token
HAVING COUNT(*) > 1;

INSERT INTO company_boards (company_id, source, board_token, status, verification_method, verified_at)
SELECT DISTINCT ON (c.ats_source, c.ats_token) c.id, c.ats_source, c.ats_token,
       'verified', 'legacy_import', NOW()
FROM companies c
WHERE c.ats_source IS NOT NULL AND c.ats_token IS NOT NULL
ORDER BY c.ats_source, c.ats_token, c.first_seen_at, c.id;

-- +goose Down
DROP TABLE board_backfill_issues;
DROP TABLE company_boards;

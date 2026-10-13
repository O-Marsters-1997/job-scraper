-- +goose Up
ALTER TABLE companies ADD COLUMN kind TEXT NOT NULL DEFAULT 'unclassified'
    CONSTRAINT companies_kind_check
    CHECK (kind IN ('unclassified', 'likely_agency', 'likely_direct', 'agency', 'direct'));

UPDATE companies SET kind = 'direct'
WHERE ats_source IS NOT NULL
   OR EXISTS (SELECT 1 FROM company_boards cb WHERE cb.company_id = companies.id);

-- +goose Down
ALTER TABLE companies DROP COLUMN kind;

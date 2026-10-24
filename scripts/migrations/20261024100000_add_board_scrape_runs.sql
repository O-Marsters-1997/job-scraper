-- +goose Up
ALTER TABLE scrape_runs
    DROP CONSTRAINT scrape_runs_kind_check,
    DROP CONSTRAINT scrape_runs_subject_check,
    ADD COLUMN board_id UUID REFERENCES company_boards(id) ON DELETE CASCADE,
    ADD COLUMN removed  INT  NOT NULL DEFAULT 0,
    ADD CONSTRAINT scrape_runs_kind_check CHECK (kind IN ('search', 'board')),
    ADD CONSTRAINT scrape_runs_subject_check CHECK (
        (kind = 'search' AND source_target_id IS NOT NULL AND board_id IS NULL)
        OR (kind = 'board' AND board_id IS NOT NULL));
CREATE INDEX scrape_runs_board_idx ON scrape_runs (board_id, started_at DESC) WHERE board_id IS NOT NULL;

-- +goose Down
DROP INDEX scrape_runs_board_idx;
DELETE FROM scrape_runs WHERE kind = 'board';
ALTER TABLE scrape_runs
    DROP CONSTRAINT scrape_runs_subject_check,
    DROP CONSTRAINT scrape_runs_kind_check,
    DROP COLUMN removed,
    DROP COLUMN board_id,
    ADD CONSTRAINT scrape_runs_kind_check CHECK (kind IN ('search')),
    ADD CONSTRAINT scrape_runs_subject_check CHECK (kind <> 'search' OR source_target_id IS NOT NULL);

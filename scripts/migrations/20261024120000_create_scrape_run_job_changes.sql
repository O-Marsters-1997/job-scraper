-- +goose Up
ALTER TABLE scrape_runs ADD COLUMN reopened INT NOT NULL DEFAULT 0;
CREATE TABLE scrape_run_job_changes (
    run_id   UUID  NOT NULL REFERENCES scrape_runs(id) ON DELETE CASCADE,
    job_id   UUID  NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    change   TEXT  NOT NULL CONSTRAINT scrape_run_job_changes_change_check
        CHECK (change IN ('new', 'removed', 'reopened')),
    snapshot JSONB NOT NULL,
    PRIMARY KEY (run_id, job_id)
);
CREATE INDEX scrape_run_job_changes_job_idx ON scrape_run_job_changes (job_id);

-- +goose Down
DROP TABLE scrape_run_job_changes;
ALTER TABLE scrape_runs DROP COLUMN reopened;

-- +goose Up
ALTER TABLE company_boards ADD COLUMN superseded_at TIMESTAMPTZ;

CREATE TABLE board_poll_state (
    board_id UUID PRIMARY KEY REFERENCES company_boards(id) ON DELETE CASCADE,
    last_completed_at TIMESTAMPTZ,
    last_scheduled_at TIMESTAMPTZ,
    last_started_at TIMESTAMPTZ,
    last_snapshot_version BIGINT NOT NULL DEFAULT 0,
    consecutive_complete_empty INT NOT NULL DEFAULT 0,
    consecutive_failures INT NOT NULL DEFAULT 0,
    lease_owner TEXT,
    lease_until TIMESTAMPTZ,
    next_due_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE board_job_observations (
    board_id UUID NOT NULL REFERENCES company_boards(id) ON DELETE CASCADE,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_snapshot_version BIGINT NOT NULL,
    PRIMARY KEY (board_id, job_id)
);

UPDATE jobs j SET primary_board_id = b.id, company_id = b.company_id
FROM company_boards b
WHERE j.primary_board_id IS NULL AND j.source = b.source
  AND j.company_slug = b.board_token AND b.status = 'verified';

INSERT INTO board_job_observations (board_id, job_id, last_seen_at, last_snapshot_version)
SELECT primary_board_id, id, scraped_at, 0 FROM jobs WHERE primary_board_id IS NOT NULL;

CREATE INDEX board_poll_state_due_idx ON board_poll_state(next_due_at) WHERE lease_until IS NULL;
CREATE INDEX board_job_observations_version_idx ON board_job_observations(board_id, last_snapshot_version);

-- +goose Down
DROP TABLE board_job_observations;
DROP TABLE board_poll_state;
ALTER TABLE company_boards DROP COLUMN superseded_at;

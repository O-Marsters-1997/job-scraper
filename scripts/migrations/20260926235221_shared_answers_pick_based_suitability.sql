-- +goose Up

CREATE TABLE option_answers (
    job_id        UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    fingerprint   TEXT NOT NULL,
    question_hash TEXT NOT NULL,
    model         TEXT NOT NULL,
    p_yes         REAL NOT NULL,
    p_no          REAL NOT NULL,
    p_not_stated  REAL NOT NULL,
    confidence    REAL NOT NULL,
    answered_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (job_id, fingerprint, question_hash, model)
);

ALTER TABLE search_config
    ADD COLUMN preferences JSONB NOT NULL DEFAULT '{}',
    DROP COLUMN scoring_questions,
    DROP COLUMN excluded_seniority;

ALTER TABLE job_scores
    ADD COLUMN breakdown JSONB NOT NULL DEFAULT '[]',
    DROP COLUMN criteria,
    DROP COLUMN confidence,
    DROP COLUMN score_config_version;

DELETE FROM job_scores;

DROP INDEX effect_outbox_user_job_idx;

ALTER TABLE effect_outbox
    DROP COLUMN user_id CASCADE,
    DROP COLUMN config_version CASCADE;

DELETE FROM effect_outbox;

CREATE UNIQUE INDEX effect_outbox_pending_idx ON effect_outbox (job_id, fingerprint, model)
    WHERE status IN ('pending', 'running');

ALTER TABLE effect_outbox ALTER COLUMN model SET DEFAULT 'typesafe/jev-1.13';

-- Queue an answer effect for every non-closed job that has an interested
-- user, so the first tick rebuilds every score under the new formula.
INSERT INTO effect_outbox (job_id, fingerprint)
SELECT j.id, j.content_fingerprint
FROM jobs j
WHERE j.closed_at IS NULL AND j.content_fingerprint IS NOT NULL
    AND (
        EXISTS (
            SELECT 1 FROM tracked_companies tc JOIN companies c ON c.id = tc.company_id
            WHERE tc.enabled AND (c.id = j.company_id OR c.slug = j.company_slug)
        )
        OR EXISTS (
            SELECT 1 FROM source_targets st
            WHERE st.enabled AND st.source = j.source
                AND (j.source IN ('wis', 'linkedin', 'indeed', 'remoteok', 'remotive') OR st.value = j.company_slug)
        )
    )
ON CONFLICT DO NOTHING;

-- +goose Down

ALTER TABLE effect_outbox ALTER COLUMN model DROP DEFAULT;
DROP INDEX effect_outbox_pending_idx;

ALTER TABLE effect_outbox
    ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    ADD COLUMN config_version TIMESTAMPTZ;

CREATE INDEX effect_outbox_user_job_idx ON effect_outbox (user_id, job_id);

ALTER TABLE job_scores
    ADD COLUMN criteria JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN confidence REAL,
    ADD COLUMN score_config_version TIMESTAMPTZ,
    DROP COLUMN breakdown;

ALTER TABLE search_config
    ADD COLUMN scoring_questions JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN excluded_seniority TEXT[] NOT NULL DEFAULT '{}',
    DROP COLUMN preferences;

DROP TABLE option_answers;

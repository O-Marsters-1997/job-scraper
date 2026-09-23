-- +goose Up
CREATE TABLE job_candidates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    normalized_url TEXT NOT NULL UNIQUE,
    source TEXT NOT NULL,
    card_title TEXT NOT NULL DEFAULT '',
    card_company TEXT NOT NULL DEFAULT '',
    card_location TEXT NOT NULL DEFAULT '',
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '60 days',
    detail_state TEXT NOT NULL DEFAULT 'unrequested' CHECK (detail_state IN ('unrequested', 'pending'))
);

CREATE INDEX job_candidates_expires_at_idx ON job_candidates(expires_at);

CREATE TABLE candidate_discoveries (
    candidate_id UUID NOT NULL REFERENCES job_candidates(id) ON DELETE CASCADE,
    source_target_id UUID NOT NULL REFERENCES source_targets(id) ON DELETE CASCADE,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (candidate_id, source_target_id)
);

CREATE INDEX candidate_discoveries_target_idx ON candidate_discoveries(source_target_id, candidate_id);

CREATE TABLE candidate_assessments (
    candidate_id UUID NOT NULL REFERENCES job_candidates(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    search_config_version TIMESTAMPTZ NOT NULL,
    relevance BOOLEAN NOT NULL,
    evaluated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (candidate_id, user_id)
);

-- +goose Down
DROP TABLE candidate_assessments;
DROP TABLE candidate_discoveries;
DROP TABLE job_candidates;

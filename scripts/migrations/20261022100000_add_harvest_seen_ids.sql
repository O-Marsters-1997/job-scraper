-- +goose Up
CREATE TABLE harvest_seen_ids (
    source        TEXT        NOT NULL,
    external_id   TEXT        NOT NULL,
    outcome       TEXT        NOT NULL CONSTRAINT harvest_seen_ids_outcome_check CHECK (outcome IN ('backfill', 'ineligible', 'exported')),
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (source, external_id)
);

-- +goose Down
DROP TABLE harvest_seen_ids;

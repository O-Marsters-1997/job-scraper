-- +goose Up
CREATE TABLE scrape_runs (
    id               UUID        PRIMARY KEY,
    kind             TEXT        NOT NULL CONSTRAINT scrape_runs_kind_check CHECK (kind IN ('search')),
    source           TEXT        NOT NULL,
    source_target_id UUID        REFERENCES source_targets(id) ON DELETE CASCADE,
    trigger          TEXT        NOT NULL CONSTRAINT scrape_runs_trigger_check CHECK (trigger IN ('scheduled', 'manual')),
    started_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at      TIMESTAMPTZ,
    stop_reason      TEXT        CONSTRAINT scrape_runs_stop_reason_check CHECK (stop_reason IN ('last_page', 'page_cap', 'failed')),
    pages            INT         NOT NULL DEFAULT 0,
    listings         INT         NOT NULL DEFAULT 0,
    new_listings     INT         NOT NULL DEFAULT 0,
    elsewhere        INT         NOT NULL DEFAULT 0,
    repeat_listings  INT         NOT NULL DEFAULT 0,
    deferred         INT         NOT NULL DEFAULT 0,
    retries          INT         NOT NULL DEFAULT 0,
    CONSTRAINT scrape_runs_finished_check CHECK ((finished_at IS NULL) = (stop_reason IS NULL)),
    CONSTRAINT scrape_runs_subject_check CHECK (kind <> 'search' OR source_target_id IS NOT NULL)
);
CREATE INDEX scrape_runs_started_idx ON scrape_runs (started_at DESC, id DESC);
CREATE INDEX scrape_runs_target_idx ON scrape_runs (source_target_id, started_at DESC) WHERE source_target_id IS NOT NULL;

CREATE TABLE scrape_run_errors (
    id      BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    run_id  UUID        NOT NULL REFERENCES scrape_runs(id) ON DELETE CASCADE,
    page    INT         NOT NULL,
    attempt INT         NOT NULL,
    message TEXT        NOT NULL,
    at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX scrape_run_errors_run_idx ON scrape_run_errors (run_id);

-- +goose Down
DROP TABLE scrape_run_errors;
DROP TABLE scrape_runs;

-- +goose Up
CREATE TABLE positions (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    employer   TEXT        NOT NULL,
    title      TEXT        NOT NULL,
    start_date DATE,
    end_date   DATE,
    sort_order INT         NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX positions_user_sort_idx ON positions (user_id, sort_order);

CREATE TABLE achievements (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    position_id UUID        NOT NULL REFERENCES positions(id) ON DELETE CASCADE,
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    text        TEXT        NOT NULL,
    sort_order  INT         NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX achievements_position_sort_idx ON achievements (position_id, sort_order);

-- +goose Down
DROP TABLE achievements;
DROP TABLE positions;

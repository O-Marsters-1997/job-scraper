-- +goose Up
CREATE TABLE company_favourites (
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    company_id UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, company_id)
);

-- +goose Down
DROP TABLE company_favourites;

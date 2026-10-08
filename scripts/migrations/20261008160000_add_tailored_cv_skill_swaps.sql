-- +goose Up
ALTER TABLE tailored_cvs ADD COLUMN skill_swaps JSONB NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE tailored_cvs DROP COLUMN skill_swaps;

-- +goose Up
ALTER TABLE achievement_outcomes ADD COLUMN variant_id UUID REFERENCES achievement_variants(id) ON DELETE SET NULL;
CREATE INDEX achievement_outcomes_variant_idx ON achievement_outcomes (variant_id) WHERE variant_id IS NOT NULL;

-- +goose Down
DROP INDEX achievement_outcomes_variant_idx;
ALTER TABLE achievement_outcomes DROP COLUMN variant_id;

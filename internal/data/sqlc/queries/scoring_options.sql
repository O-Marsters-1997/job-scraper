-- name: ListScoringOptions :many
SELECT * FROM scoring_options ORDER BY dimension, id;

-- name: InsertScoringOption :exec
INSERT INTO scoring_options (id, dimension, label, question)
VALUES (sqlc.arg(id)::text, sqlc.arg(dimension)::scoring_dimension, sqlc.arg(label)::text, sqlc.arg(question)::text);

-- name: RewordScoringOption :execrows
UPDATE scoring_options SET question = sqlc.arg(question)::text WHERE id = sqlc.arg(id)::text;

-- name: RetireScoringOption :execrows
UPDATE scoring_options SET retired_at = NOW() WHERE id = $1 AND retired_at IS NULL;

-- name: ListScoringOptions :many
SELECT * FROM scoring_options ORDER BY dimension, id;

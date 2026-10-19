-- name: GetLetterSkeleton :one
SELECT user_id, intro, conclusion, sections, updated_at
FROM letter_skeletons WHERE user_id = $1;

-- name: UpsertLetterSkeleton :one
INSERT INTO letter_skeletons (user_id, intro, conclusion, sections)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id)
DO UPDATE SET intro = EXCLUDED.intro, conclusion = EXCLUDED.conclusion,
              sections = EXCLUDED.sections, updated_at = NOW()
RETURNING user_id, intro, conclusion, sections, updated_at;

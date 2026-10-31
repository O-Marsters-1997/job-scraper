-- +goose Up
ALTER TABLE score_feedback DROP CONSTRAINT score_feedback_direction_check;
ALTER TABLE score_feedback ADD CONSTRAINT score_feedback_direction_check CHECK (direction IN ('higher', 'lower', 'accurate'));
ALTER TABLE score_feedback DROP CONSTRAINT score_feedback_reason_check;
ALTER TABLE score_feedback ADD CONSTRAINT score_feedback_reason_check CHECK (kind = 'job' OR btrim(reason) <> '');

-- +goose Down
DELETE FROM score_feedback WHERE direction = 'accurate' OR (kind = 'job' AND btrim(reason) = '');
ALTER TABLE score_feedback DROP CONSTRAINT score_feedback_reason_check;
ALTER TABLE score_feedback ADD CONSTRAINT score_feedback_reason_check CHECK (btrim(reason) <> '');
ALTER TABLE score_feedback DROP CONSTRAINT score_feedback_direction_check;
ALTER TABLE score_feedback ADD CONSTRAINT score_feedback_direction_check CHECK (direction IN ('higher', 'lower'));

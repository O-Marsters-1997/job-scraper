-- +goose Up
ALTER TABLE option_answers ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE option_answers DROP CONSTRAINT option_answers_pkey;

INSERT INTO option_answers (user_id, job_id, fingerprint, question_hash, model, p_yes, p_no, p_not_stated, confidence, answered_at)
SELECT s.user_id, a.job_id, a.fingerprint, a.question_hash, a.model, a.p_yes, a.p_no, a.p_not_stated, a.confidence, a.answered_at
FROM option_answers a
JOIN job_scores s ON s.job_id = a.job_id AND s.score_fingerprint = a.fingerprint
WHERE a.user_id IS NULL;

DELETE FROM option_answers WHERE user_id IS NULL;

ALTER TABLE option_answers ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE option_answers ADD PRIMARY KEY (user_id, job_id, fingerprint, question_hash, model);
CREATE INDEX option_answers_job_id_idx ON option_answers (job_id);

-- +goose Down
DROP INDEX option_answers_job_id_idx;
ALTER TABLE option_answers DROP CONSTRAINT option_answers_pkey;

DELETE FROM option_answers a
USING option_answers b
WHERE a.job_id = b.job_id AND a.fingerprint = b.fingerprint AND a.question_hash = b.question_hash
    AND a.model = b.model AND a.user_id > b.user_id;

ALTER TABLE option_answers DROP COLUMN user_id;
ALTER TABLE option_answers ADD PRIMARY KEY (job_id, fingerprint, question_hash, model);

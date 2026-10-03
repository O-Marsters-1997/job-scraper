-- +goose Up
UPDATE scoring_options SET question = 'Is the role pitched at Junior or graduate level (not Mid, Senior or above)?' WHERE id = 'seniority:junior';
UPDATE scoring_options SET question = 'Is the role pitched at Mid level (not Junior, Senior, Staff, Principal, Lead or Head of)?' WHERE id = 'seniority:mid';
UPDATE scoring_options SET question = 'Is the role pitched at Senior level (not Staff, Principal, Lead or Head of)?' WHERE id = 'seniority:senior';
UPDATE scoring_options SET question = 'Is the role pitched at Staff level or above (Staff, Principal, Lead or Head of)?' WHERE id = 'seniority:staff';
UPDATE scoring_options SET question = 'Is the role fully remote (not hybrid or office-based)?' WHERE id = 'work:remote';
UPDATE scoring_options SET question = 'Is the role hybrid, split between home and an office (not fully remote or fully onsite)?' WHERE id = 'work:hybrid';
UPDATE scoring_options SET question = 'Is the role fully onsite in an office (not remote or hybrid)?' WHERE id = 'work:onsite';
UPDATE scoring_options SET question = 'Is the company at Seed stage (not pre-seed, Series A or later, or public)?' WHERE id = 'stage:seed';
UPDATE scoring_options SET question = 'Is the company at Series A stage (not Seed, Series B or later, or public)?' WHERE id = 'stage:series_a';
UPDATE scoring_options SET question = 'Is the company at Series B stage (not Series A, Series C or later, or public)?' WHERE id = 'stage:series_b';
UPDATE scoring_options SET question = 'Is the company at Series C or later stage (not Seed, Series A or B, or public)?' WHERE id = 'stage:series_c_plus';
UPDATE scoring_options SET question = 'Is the company publicly listed (not a private or venture-backed company)?' WHERE id = 'stage:public';
UPDATE scoring_options SET question = 'Is the company''s main product or market ' || label || '? Employee benefits, perks and clients'' industries don''t count.' WHERE dimension = 'domain';

-- +goose Down
SELECT 1;

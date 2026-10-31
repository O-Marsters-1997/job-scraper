-- +goose Up
UPDATE scoring_options SET question = 'Is the company''s main product or market healthcare or healthtech (medical care, health services, medical devices)? Employee benefits, perks and clients'' industries don''t count.' WHERE id = 'domain:health';
UPDATE scoring_options SET question = 'Is the company''s main product or market climate tech (decarbonisation, carbon accounting, clean technology)? Employee benefits, perks and clients'' industries don''t count.' WHERE id = 'domain:climate';
UPDATE scoring_options SET question = 'Is the company''s main product or market the energy sector (power generation, supply or grid)? Employee benefits, perks and clients'' industries don''t count.' WHERE id = 'domain:energy';
UPDATE scoring_options SET question = 'Is the company''s main product or market a consultancy or services agency that builds products for clients? Employee benefits, perks and clients'' industries don''t count.' WHERE id = 'domain:agency';
UPDATE scoring_options SET question = 'Is the company''s main product or market legal tech: software or services sold to lawyers or law firms? Employee benefits, perks and clients'' industries don''t count.' WHERE id = 'domain:legal';
UPDATE scoring_options SET question = 'Is the company''s main product or market media or publishing (news, streaming, content)? Employee benefits, perks and clients'' industries don''t count.' WHERE id = 'domain:media';
UPDATE scoring_options SET question = 'Is the company''s main product or market cybersecurity (products that protect systems, data or identities)? Employee benefits, perks and clients'' industries don''t count.' WHERE id = 'domain:security';

-- +goose Down
SELECT 1;

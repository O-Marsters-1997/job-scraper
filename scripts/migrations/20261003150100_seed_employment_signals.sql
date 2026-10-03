-- +goose Up
INSERT INTO scoring_options (id, dimension, label, question) VALUES
    ('employment:permanent', 'employment', 'Permanent', 'Is the role a permanent, full-time employment contract (not contract, freelance or part-time)?'),
    ('employment:contract', 'employment', 'Contract', 'Is the role a fixed-term or day-rate contract, including outside IR35 or inside IR35 (not permanent)?'),
    ('employment:part_time', 'employment', 'Part-time', 'Is the role part-time (not full-time)?'),
    ('role:product-engineering', 'role', 'Product engineering', 'Is this a product engineering role: shipping user-facing features across the stack?'),
    ('domain:recruitment', 'domain', 'recruitment', 'Is the company a recruitment or staffing agency, hiring on behalf of an unnamed client?')
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM scoring_options WHERE id IN ('employment:permanent', 'employment:contract', 'employment:part_time', 'role:product-engineering', 'domain:recruitment');

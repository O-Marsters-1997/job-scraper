-- name: BackfillCompanyJobFingerprints :exec
UPDATE jobs j SET content_fingerprint = encode(sha256(convert_to(
    replace(replace(replace(replace(replace(to_json(ARRAY[
        j.title, j.description, j.location, j.salary_raw, j.work_arrangement
    ])::text,
    '&', chr(92) || 'u0026'), '<', chr(92) || 'u003c'),
    '>', chr(92) || 'u003e'), chr(8232), chr(92) || 'u2028'),
    chr(8233), chr(92) || 'u2029'), 'UTF8')), 'hex')
FROM companies c
WHERE c.id = $1::uuid AND (j.company_id = c.id OR j.company_slug = c.slug)
    AND j.closed_at IS NULL AND j.content_fingerprint IS NULL;

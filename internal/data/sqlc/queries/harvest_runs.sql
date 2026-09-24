-- name: GetHarvestRun :one
SELECT last_succeeded_at FROM harvest_runs WHERE harvester = $1;

-- name: SetHarvestRun :exec
INSERT INTO harvest_runs (harvester, last_succeeded_at) VALUES ($1, NOW())
ON CONFLICT (harvester) DO UPDATE SET last_succeeded_at = NOW();

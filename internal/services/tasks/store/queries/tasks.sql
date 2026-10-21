-- name: SeedDefaultTaskTypes :exec
INSERT INTO task_types (user_id, name, colour)
VALUES
    ($1, 'Apply',     '#6366f1'),
    ($1, 'Follow up', '#c2410c'),
    ($1, 'Reach out', '#15803d'),
    ($1, 'Prep',      '#64748b');

-- name: ListTaskTypes :many
SELECT tt.id, tt.name, tt.colour, tt.created_at,
    (SELECT count(*) FROM tasks t WHERE t.type_id = tt.id)::int AS task_count
FROM task_types tt
WHERE tt.user_id = $1
ORDER BY tt.created_at ASC, tt.name ASC;

-- name: CreateTaskType :one
INSERT INTO task_types (user_id, name, colour)
VALUES ($1, $2, $3)
RETURNING id, name, colour, created_at, 0::int AS task_count;

-- name: UpdateTaskType :one
UPDATE task_types tt
SET name = sqlc.arg(name), colour = sqlc.arg(colour)
WHERE tt.id = sqlc.arg(id) AND tt.user_id = sqlc.arg(user_id)
RETURNING tt.id, tt.name, tt.colour, tt.created_at,
    (SELECT count(*) FROM tasks t WHERE t.type_id = tt.id)::int AS task_count;

-- name: DeleteTaskType :execrows
DELETE FROM task_types WHERE id = $1 AND user_id = $2;

-- name: PageTasks :many
SELECT
    t.id, t.type_id, t.job_id, t.link, t.notes, t.completed_at, t.created_at, t.updated_at,
    tt.name   AS type_name,
    tt.colour AS type_colour,
    j.title        AS job_title,
    j.company_slug AS job_company_slug,
    COALESCE(j.posted_at, j.posted_after, j.first_discovered_at) AS job_posted_at
FROM tasks t
JOIN task_types tt ON tt.id = t.type_id
LEFT JOIN jobs j ON j.id = t.job_id
WHERE t.user_id = sqlc.arg(user_id)::uuid
  AND (sqlc.narg(type_id)::uuid IS NULL OR t.type_id = sqlc.narg(type_id)::uuid)
  AND (sqlc.narg(job_id)::uuid IS NULL OR t.job_id = sqlc.narg(job_id)::uuid)
  AND (t.completed_at IS NOT NULL) = sqlc.arg(done)::bool
ORDER BY
    CASE WHEN sqlc.arg(sort)::text = 'job_posted' THEN COALESCE(j.posted_at, j.posted_after, j.first_discovered_at) END DESC NULLS LAST,
    t.created_at DESC,
    t.id
LIMIT sqlc.arg(page_limit)::int OFFSET sqlc.arg(page_offset)::int;

-- name: CountTasks :one
SELECT count(*)
FROM tasks t
WHERE t.user_id = sqlc.arg(user_id)::uuid
  AND (sqlc.narg(type_id)::uuid IS NULL OR t.type_id = sqlc.narg(type_id)::uuid)
  AND (sqlc.narg(job_id)::uuid IS NULL OR t.job_id = sqlc.narg(job_id)::uuid)
  AND (t.completed_at IS NOT NULL) = sqlc.arg(done)::bool;

-- name: CreateTask :one
WITH created AS (
    INSERT INTO tasks (user_id, type_id, job_id, link, notes)
    SELECT sqlc.arg(user_id)::uuid, tt.id, sqlc.narg(job_id)::uuid, sqlc.narg(link)::text, sqlc.narg(notes)::text
    FROM task_types tt
    WHERE tt.id = sqlc.arg(type_id)::uuid AND tt.user_id = sqlc.arg(user_id)::uuid
    RETURNING id, type_id, job_id, link, notes, completed_at, created_at, updated_at
)
SELECT
    t.id, t.type_id, t.job_id, t.link, t.notes, t.completed_at, t.created_at, t.updated_at,
    tt.name   AS type_name,
    tt.colour AS type_colour,
    j.title        AS job_title,
    j.company_slug AS job_company_slug,
    COALESCE(j.posted_at, j.posted_after, j.first_discovered_at) AS job_posted_at
FROM created t
JOIN task_types tt ON tt.id = t.type_id
LEFT JOIN jobs j ON j.id = t.job_id;

-- name: SetTaskDone :one
WITH updated AS (
    UPDATE tasks
    SET completed_at = CASE WHEN sqlc.arg(done)::bool THEN COALESCE(completed_at, sqlc.arg(now)::timestamptz) END,
        updated_at = sqlc.arg(now)::timestamptz
    WHERE id = sqlc.arg(id)::uuid AND user_id = sqlc.arg(user_id)::uuid
    RETURNING id, type_id, job_id, link, notes, completed_at, created_at, updated_at
)
SELECT
    t.id, t.type_id, t.job_id, t.link, t.notes, t.completed_at, t.created_at, t.updated_at,
    tt.name   AS type_name,
    tt.colour AS type_colour,
    j.title        AS job_title,
    j.company_slug AS job_company_slug,
    COALESCE(j.posted_at, j.posted_after, j.first_discovered_at) AS job_posted_at
FROM updated t
JOIN task_types tt ON tt.id = t.type_id
LEFT JOIN jobs j ON j.id = t.job_id;

-- name: UpdateTask :one
WITH updated AS (
    UPDATE tasks
    SET type_id = tt.id,
        link = sqlc.narg(link)::text,
        notes = sqlc.narg(notes)::text,
        updated_at = sqlc.arg(now)::timestamptz
    FROM task_types tt
    WHERE tasks.id = sqlc.arg(id)::uuid AND tasks.user_id = sqlc.arg(user_id)::uuid
      AND tt.id = sqlc.arg(type_id)::uuid AND tt.user_id = sqlc.arg(user_id)::uuid
    RETURNING tasks.id, tasks.type_id, tasks.job_id, tasks.link, tasks.notes, tasks.completed_at, tasks.created_at, tasks.updated_at
)
SELECT
    t.id, t.type_id, t.job_id, t.link, t.notes, t.completed_at, t.created_at, t.updated_at,
    tt.name   AS type_name,
    tt.colour AS type_colour,
    j.title        AS job_title,
    j.company_slug AS job_company_slug,
    COALESCE(j.posted_at, j.posted_after, j.first_discovered_at) AS job_posted_at
FROM updated t
JOIN task_types tt ON tt.id = t.type_id
LEFT JOIN jobs j ON j.id = t.job_id;

-- name: LockTask :one
SELECT t.job_id, t.completed_at, (tt.name = 'Apply')::bool AS is_apply
FROM tasks t
JOIN task_types tt ON tt.id = t.type_id
WHERE t.id = sqlc.arg(id)::uuid AND t.user_id = sqlc.arg(user_id)::uuid
FOR UPDATE OF t;

-- name: CreateApplyTask :exec
INSERT INTO tasks (user_id, type_id, job_id)
SELECT sqlc.arg(user_id)::uuid, tt.id, sqlc.arg(job_id)::uuid
FROM task_types tt
WHERE tt.user_id = sqlc.arg(user_id)::uuid AND tt.name = 'Apply'
  AND NOT EXISTS (
      SELECT 1 FROM tasks t
      WHERE t.type_id = tt.id AND t.job_id = sqlc.arg(job_id)::uuid AND t.completed_at IS NULL
  );

-- name: CompleteApplyTasks :exec
UPDATE tasks t
SET completed_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now)::timestamptz
FROM task_types tt
WHERE tt.id = t.type_id AND tt.name = 'Apply'
  AND t.user_id = sqlc.arg(user_id)::uuid AND t.job_id = sqlc.arg(job_id)::uuid
  AND t.completed_at IS NULL;

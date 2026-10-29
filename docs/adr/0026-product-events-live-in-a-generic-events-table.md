# ADR 0026 — Product events live in a generic events table

Prompted by #617. Scoring needs labelled signal tied to the score a User reacted to: a dismissed job was
scored too high, an applied job about right or too low. `job_scores` keeps one row per (Job, User) and
overwrites it on every rescore, so a later join returns today's score, not the one the User saw.

## Decision

- **One `events` table, owned by the `events` context.** Columns: `id`, `user_id`, `type`, `subject_id`,
  `props` (jsonb) and `created_at`. Company and application events fit later without a new table.
- **`type` is `text` with a named CHECK constraint** (`events_type_check`), per the postgres-best-practices
  skill (status columns). Adding or dropping a value is a one-statement constraint swap in an ordinary
  migration. The other closed sets, `user_role` and `scoring_dimension`, follow the same shape. Go gets
  typed sets from sqlc overrides, so `events.Type` and its siblings stay compile-time checked.
- **`subject_id` has no foreign key.** The type decides what it points at. The log outlives the row.
- **`props` is one Go struct per type, validated on write.** Job and application events carry the score
  snapshot (`score`, `band`, `score_model`, `score_fingerprint`, `breakdown`), taken server side from
  scoring through the `ScoreSnapshotter` interface and omitted when the job has no score. The `breakdown`
  is stored only on `job_dismissed` and application events; `job_opened` and `alert_opened` are high volume
  and keep the scalar fields. Application events read the snapshot before their transaction opens, so a
  request never holds two pool connections.
- **Frontend events** (`job_opened`, `job_dismissed`, `alert_opened`) go through `POST /events`, fire and
  forget. **Application events** are written by `applications` in its own transaction through the events
  module's ports.

## Consequences

- ADR 0010's events-from-logs stays operational telemetry. Product events live in Postgres.
- Impressions are recorded as `job_impression` (see below), not as raw renders.
- Explicit labels stay in `job_grades`, `score_feedback` and `answer_corrections`. Training queries
  SELECT-join them with `events`.

## Job impressions

`job_impression` records a job shown to a User, so training can tell "not clicked" from "not seen".
Clients post a batch (1 to 50 items) to `POST /events/impressions`: surface (`triage` or `list`), sort,
and a 1-based position per job. The server writes one row per item with the scalar score snapshot and
`props.surface`, `props.sort` and `props.position`; it is server-only on `POST /events`. The web Jobs
list sends one batch per loaded page and sort, so refreshes and re-renders of the same page add nothing.
Alert impressions are not recorded.

## Dismiss reasons and undo

A dismissal's reason chips are picked after the `job_dismissed` event fires, so
the event carries no reason. Training reads the reason from `job_grades.reasons`.
Undoing a dismissal records a second `job_dismissed` event with `props.undone = true`.

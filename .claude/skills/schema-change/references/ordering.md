# Migration ordering across branches

Goose applies migrations in the order of their filename's timestamp prefix
(`YYYYMMDDHHMMSS_name.sql`), not the order they were written or merged. If
your branch and another branch each add a migration off the same base
commit, the merged history's timestamp order may not match either author's
intent — goose only knows the numbers in the filenames, not which PR merged
first or which migration logically depends on which.

This has actually happened in this repo. Two commits re-timestamped a
migration after the fact, purely to fix ordering relative to another one
that had landed on `main` in between:

- `ef5e4cb` "Order scoring migration after board polling" — renamed
  `20260923000005_scoring_outbox.sql` to `20260923000011_scoring_outbox.sql`.
- `0766f91` "Order pagination migration after scoring" — renamed
  `20260923000005_page_jobs.sql` to `20260923000012_page_jobs.sql`.

Both were plain `git mv`s (file content untouched) to a later timestamp —
not edits to a migration that had already been applied against a shared
database. That distinction matters: the "never edit an applied migration"
rule (`AGENTS.md` "Gotchas") is about not rewriting history that a real
database has already run. Renaming an *unapplied* migration you're still
iterating on before merge — because it turns out to depend on something
that landed after you branched — isn't the same violation.

**Rule:** if your migration's logic depends on a table or column another
migration introduces, and that other migration's timestamp sorts after
yours once both are merged, rename your file to a fresh timestamp that
sorts after it. Check with `just migrate-status` and `git log --oneline -1
-- scripts/migrations/<the-one-you-depend-on>` to see what's already on
`main`. Only rename a migration your own branch introduced and hasn't
been applied anywhere shared yet — never a migration someone else's merged
branch already ran.

# ADR 0034 — Comments are separate from Notes

Status: accepted. Supersedes the "comments are notes with a quote" model in `docs/prds/job-notes.md` (J2, J3).

## Context

#1026 shipped comments as `job_notes` rows with a non-null `quote`. They share the Notes list and the
sidebar composer. A feedback session on 2026-10-18 found this unusable. A comment reads as a scrap of text
in the notes list, cut off from the passage it is about. The user wants Google Docs / Claude Docs
behaviour.

## Decision

- **A Comment is its own entity.** It lives in a new `job_comments` table owned by `applications`
  (ADR 0011). It belongs to (user, Job) and is anchored by the same `quote` exact/prefix/suffix as today.
  `job_notes` loses its quote columns. Existing quoted notes are migrated into `job_comments`.
- **Comments live on the description, never in the Notes list.** Selecting text shows an icon button (no
  "Comment" label) that opens an inline composer next to the selection. A saved comment shows a muted
  outline or underline on its passage. Hover or click opens a popover with the comment and edit/delete.
- **One comment per anchor, no threads, no resolve.** Add these if a real need appears.
- **Notes stay plain text** (no markdown, as the PRD says). They move from the sidebar card into a
  Description | Notes tab on the job page and also show on the application details page.

## Consequences

Notes and Comments get separate endpoints, hooks and types. `lib/noteAnchor.ts` and
`lib/noteHighlights.ts` now serve Comments. An orphaned comment, whose passage is no longer in the
description, still needs a place to show. It is listed under the description rather than dropped.

# ADR 0037 — iOS Shortlisted list and search on Due

Status: accepted. Agreed in a grilling session on 2026-10-27. Amends [ADR 0035](0035-ios-companion-is-a-triage-app.md).

## Context

ADR 0035 gives the phone two tabs, Triage and Due, and says any other screen needs a new decision. #1394 put
an apply row on Due for every Shortlisted job without a CV draft. That crowded Due, so the owner reverted
it: Due should only hold what's actually due. But a Shortlist made without "Draft CV" then has no way back
in on the phone, and neither list can be searched.

## Decision

- **Due** holds chases due, open Tasks and CV drafts only. Jobs that only have an open `apply` Task leave Due.
- **Shortlisted list.** A screen pushed from Due. It lists Shortlisted jobs with an open `apply` Task, no CV
  draft and no `finish` Task, highest score first.
  - Entry is a "Not drafted · N ›" row at the foot of the Due notebook. The row also appears in the
    "Nothing due" empty state, and is hidden when N is 0.
  - Each row offers **Draft CV** (the job moves to Due once the draft exists) and **Apply** (the
    apply-on-listing sheet Due already uses). Tapping the row opens Job detail.
  - It is built from the due payload the app already loads. There is no new endpoint, and it is online
    only, like Due.
- **Search.** Native `.searchable` on Due and on the Shortlisted list. It filters the loaded list locally
  by job title and company.
- Still out of scope: browsing or searching all jobs, and a third tab.

## Consequences

The tab bar stays at two, and Due goes back to what ADR 0035 described. A job can't fall out of reach once
Shortlisted. The DTO and the hand-mirrored Swift models don't change.

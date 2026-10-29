# ADR 0035 — The iOS app is a phone triage companion, not a port

Status: accepted. Scope agreed in a grilling session on 2026-10-25.

## Context

The web app is already an installable PWA with Web Push, and `ios/` holds a SwiftUI client with only login,
a scored jobs list and a read-only job detail. A native app has to earn its place over the PWA. The web UI
is table-first and desktop-dense (PRODUCT.md: "the table is the interface"), and that shape doesn't fit a
phone.

## Decision

- **Purpose.** The phone runs a triage workflow in 1–3 minute daily bursts (commute, a queue, between
  calls). Anything needing real reading or writing waits for the desktop.
- **Triage tab** (replaces the current Jobs list):
  - **Triage queue:** `GET /jobs/triage`: scored jobs that are unseen, not dismissed and have no Application,
    Relevant order. Once grading accuracy is trusted it
    narrows to `suit=<notify_threshold>`.
  - **Card:** title, company, location and work arrangement, salary, age, Band and score, and the three
    breakdown rows that moved the score most: gates, then the favourite row, then the rest by `ScoreRow.impact`.
  - **Right swipe = Shortlist:** creates the Application (Draft status, which also creates the `apply`
    Task). An optional "Draft CV" toggle starts a draft from the default CV (`POST /tailoring/drafts`).
  - **Left swipe = Dismiss:** Grade `no`, no reason asked.
  - **"…" menu:** Dismiss with reason (Grade reason chips), Exclude company
    (`PUT /companies/{id}/exclusion`), Add note.
  - **Skip:** scrolling past a card without choosing only marks it seen (`POST /jobs/seen`).
  - Every action has undo.
- **Implicit grading.** The phone has no Grade UI. Scoring learns from Dismiss (`no`) and Applications
  (Replay positives).
- **Offline, triage only.** The next ~50 cards are cached with their full descriptions, and triage actions
  queue while offline. Replay runs oldest-first and the last write wins: "already exists" counts as success
  and a 404 drops the action. Afterwards a short note says how many synced and how many were skipped.
- **Due list tab** (online only):
  - Chases due (done, snooze, log a Recruiter contact).
  - Open Tasks (tick done).
  - Ready CV drafts: a read-only PDF (`GET /tailoring/drafts/{id}/pdf`) that can be shared or saved to
    Files, plus **Mark applied**. Mark applied keeps the draft and ticks the job's `apply` Task, which marks
    the Application Applied in the same transaction.
- **Job detail.** Full description (sanitised HTML), full breakdown, link to the original posting, the
  triage actions, the job Note, and viewing and logging Recruiter contacts.
- **Out of scope for v1:**
  - Search Config and Picks, Grade UI, Corrections, Score Feedback.
  - CV editing, review and discard; cover letters; experience bank; CV template management.
  - Companies, Boards and Source Targets; Scrape Runs; insights; overview stats.
  - General Status editing (Mark applied is the only Status change), comments, browsing or searching all
    jobs.
  - Settings, admin, signup.
- **No native push in v1.** Email and Web Push stay the nudge. APNs and chase-reminder pushes are deferred.
- **Distribution.** Free provisioning to the owner's phone (re-signed every 7 days). A paid account and
  TestFlight come in with APNs or peers.
- **DTO drift guard.**
  - Go tests write golden JSON for each dto the app decodes, and a Swift test decodes those fixtures.
  - A macOS CI job runs only when `internal/dto/**`, the fixtures or `ios/**` change.
- **Success test after ~4 weeks.** The owner opens the app most days and the phone is the source of most
  new triage decisions. Each phone decision also posts a product event tagged `client=ios` (ADR 0026), and
  the grades and applications tables are left untouched.

## Consequences

Nearly all of v1 reuses existing endpoints. The new backend work is the iOS client tag on events and the
golden fixtures. The Swift side gains models for Application, Task, Note, Recruiter contact, Draft, CV
template and `Job.Description`, an HTML renderer and a local action queue. A screen outside this list
needs a new decision, not a quiet addition.

## Amendment (2026-10-27): Notifications tab

The app gains a third tab, **Notifications**, the in-app record of the new-match alerts the server already
pushes through ntfy (`GET /notifications`). It lists each notification
with its job summary, badges the unread count, marks a row read when tapped (which opens Job detail) and
offers "Mark all as read"; a long press previews the job description. It is a read-and-jump surface that
feeds triage, not a new workflow, so native push (APNs) stays deferred as above.

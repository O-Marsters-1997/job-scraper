# Recruiter agencies

## Problem

Most jobs in the app come from LinkedIn, and most of those are posted by recruitment agencies. I can't
tell them apart from jobs posted by the hiring company, and I have no way to track which agencies I've
spoken to without storing people's names.

## Journeys

### J1. Mark a company as a recruiter

Surface: UI

1. Open the actions menu on a job in the job list.
2. Choose "Mark recruiter".
3. Every job from that company now shows as an agency job.

→ The company's kind is `agency`, and the job leaves the Direct filter and appears under Agency.

### J2. Undo a wrong mark

Surface: UI

1. Open the actions menu on an agency job.
2. Choose "Not a recruiter".

→ The company's kind is `direct`, and all its jobs return to the Direct filter.

### J3. Look only at jobs posted by the employer

Surface: UI

1. Open the poster filter on the job list. It defaults to All.
2. Pick Direct.

→ The list shows jobs from direct, likely direct and unclassified companies only, and the URL keeps
the filter on reload.

### J4. See an agency's roles

Surface: UI

1. Open the companies list and filter it to agencies.
2. Open one agency.

→ The agency page lists that agency's open roles.

### J5. Spot a likely recruiter without marking it myself

Surface: UI

1. A new company appears on LinkedIn (or another discovery source).
2. The job list shows a "Maybe recruiter" badge on its jobs.
3. Confirm it with "Mark recruiter" or reject it with "Not a recruiter" from the actions menu.

→ The company's kind is `agency` or `direct`, the badge is gone, and the classifier never runs for that
company again.

### J6. Log that I contacted the recruiter for a role

Surface: UI

1. Open an agency job.
2. Choose "Log recruiter contact".
3. Pick a date, a channel (LinkedIn message, email, call, inbound) and an outcome (no reply, spoke,
   submitted, rejected).

→ The job shows its contact history, newest first, and a second contact on the same job adds a row
rather than replacing the first.

### J7. See which desks at an agency I haven't reached

Surface: UI

1. Open an agency page.
2. Read the rollup at the top and the contact column on each role.

→ The page shows roles total, roles with a recruiter contact, last contact date and best outcome, and
roles with no contact are visibly marked.

## Delivery slices

1. **Mark and filter**: I can mark a company as a recruiter or not, filter the job list to Direct or
   Agency, and see an agency's roles. Companies found through their own ATS board count as direct
   with no action. Journeys: J1, J2, J3, J4
2. **Likely-recruiter badge**: new discovery-source companies get classified once, and I confirm or
   reject the guess. Journeys: J5
3. **Recruiter contacts**: I log contacts against agency roles and see the per-agency rollup.
   Journeys: J6, J7

## Concepts

- **Company kind** lives on the shared `companies` table, so it's one fact for every user. States are
  `unclassified`, `likely_agency`, `likely_direct`, `agency`, `direct`. The two `likely_` states come
  from the classifier. `agency` and `direct` come from a user, and the classifier never overwrites them.
- **Filter buckets.** Agency covers `agency` and `likely_agency`. Direct covers `direct`,
  `likely_direct` and `unclassified`.
- **Badge.** Only `likely_agency` shows "Maybe recruiter". `likely_direct` shows nothing.
- **Classifier** is a separate call using the Jev model. It runs at most once per company, the first
  time a discovery source (LinkedIn, Indeed, RemoteOK, Remotive, WIS) brings that company in. Input is
  the company name and one job description, including LinkedIn's "Industries" criteria line. A failed
  call leaves the company `unclassified`, and it retries on the company's next job.
- **ATS companies** (Greenhouse, Lever, Ashby and the rest) are set to `direct` on sight. A company with
  its own board is the employer.
- **Jev fit scoring is unchanged.** Agency jobs still get scored. Only the recruiter question is
  asked once per company.
- **Recruiter Contact** is a new `recruiter_contacts` table in the `applications` context: user, job,
  contacted date, channel enum, outcome enum. Contacts attach to roles, not people. Each agency role
  usually has its own recruiter, so the role stands in for the person.
- **Ownership.** `jobsearch` owns company kind. The classifier writes it there, either from inside
  `jobsearch` or through a `jobsearch` port. The agency rollup is a `jobsearch` read that joins
  `recruiter_contacts`, a cross-context SELECT that ADR 0011 allows.

## Out of scope

- Any record of a person: names, profiles, emails, phone numbers. This replaces the person-level
  recruiter CRM proposed in `ideas/reports/2026-10-03-outreach.md`.
- Free text on a recruiter contact. With only enums and a date, the schema can't hold a name.
- A per-user agency relationship status. The rollup covers it.
- Per-user overrides of company kind.
- Backfilling companies already in the database. They stay `unclassified` (so they count as Direct)
  until a user marks them.
- A separate "job board / aggregator" kind.

## Open questions

- **Should Unknown be its own filter option?** Resolved: no. `unclassified` folds into Direct. Without
  a backfill, existing companies sit there until marked, which I accepted.
- **Does Jev skip agency jobs entirely?** Resolved: no. It skips only the recruiter check.

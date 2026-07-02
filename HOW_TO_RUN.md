# How to use job-scraper

## Adding a job source

Sources are configured via the frontend under **Settings → Sources**. Add a source by selecting the provider (Greenhouse, Lever, Ashby, etc.) and entering the board token (for ATS sources) or search URL (for LinkedIn/Indeed).

The worker picks up new sources on its next startup — restart the worker after adding one.

## Triggering a scrape

The worker scrapes automatically every hour, but only rebuilds sources for targets that are actually due (per-target `check_interval_minutes`, default 360). To see jobs flow immediately after adding a source, restart the worker — it runs the orchestrator once on boot before settling into the cron.

## Tracking a company

Under **Companies**, search the shared catalog or paste an ATS board URL to add a new one, then toggle **Track** — this enables that company's source target for you. Open a company's detail page to change its check frequency or view its jobs.

## Adding a search / filter

Searches and keyword filters are configured per-user under **Settings → Search Config**. Jobs are scored against your search config at ingest time using Claude, so only jobs matching your criteria surface.

## Verifying things work

- **API up:** `curl http://localhost:8080/jobs` → `401` (not connection refused)
- **Sources registered:** worker logs `sources built from db count=N` on startup; if `N=0`, no sources are configured
- **Jobs flowing (ATS):** jobs appear in the frontend after a scrape; API logs show `/ingest` calls
- **Jobs flowing (HTML/queue):** same as above; additionally check `just queue-list` to see URLs being enqueued before dispatch
- **Scoring working:** jobs in the frontend show a suitability score; if missing, check `ANTHROPIC_API_KEY` and `SCORING_USER_ID` are set

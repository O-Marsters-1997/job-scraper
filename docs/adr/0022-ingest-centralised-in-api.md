# ADR 0022 — Ingest centralised in the API

Persisting, scoring and notifying all happen behind `POST /ingest` in `cmd/api`. The worker only scrapes and posts jobs there, authenticated with a shared bearer token (`INGEST_SERVICE_TOKEN`, not exposed to browsers). This gives one ingest path for every job source and keeps scorer/notifier failures and credentials (Anthropic, Resend) out of the scraper process.

Trade-off: the worker needs `API_BASE_URL`, and a network partition that outlasts the exporter's bounded 5xx retries drops those jobs, which is acceptable at personal volume.

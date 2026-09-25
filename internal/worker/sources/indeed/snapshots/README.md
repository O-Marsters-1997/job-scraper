# Placeholder fixtures

`list_search.html` and `detail_job.html` (and their paired `.json`) in this
directory are **hand-built placeholders**, not real Indeed captures. This
environment had no `BRIGHTDATA_PROXY_URL` configured and a direct unproxied
fetch to indeed.com was blocked by a Cloudflare interstitial (see
`../testdata/blocked_cloudflare.html`), so no live Indeed page could be
fetched to seed these fixtures honestly.

They exist so the parser has *something* to exercise in CI, built to match
the selectors documented in `indeed.go`'s package comment (data-testid hooks
used by the public JobSpy scraper). They do not prove the parser works
against real Indeed markup.

**Regenerate against real pages once proxy access is available:**

```sh
just cli download indeed list_search <a-real-indeed-search-url>
just cli download indeed detail_job <a-real-indeed-detail-url>
just cli rebase indeed
```

Delete this README once real fixtures replace the placeholders.

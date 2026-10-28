# ADR 0033 — hiring.cafe is read through a stealth Chrome tab

Issue: [#1024](https://github.com/O-Marsters-1997/job-scraper/issues/1024). Status: superseded by
[#1250](https://github.com/O-Marsters-1997/job-scraper/issues/1250): the Source now fetches through Decodo's Web
Data API (`hiringcafe.API`, `DECODO_SCRAPE_TOKEN`), with no Chrome or tunnel. The rest is history.

## Context

hiring.cafe sits behind Cloudflare Bot Management. Live-tested on 2026-10-18: plain Go HTTP gets a 403
challenge, so the Source needs a real browser. The worker already has a residential proxy (Decodo) and a
per-host `sources.Gate`.

## Decision

- **A go-rod browser with `go-rod/stealth`**, launched lazily on first use and kept as one tab for the
  worker's life. `playwright-go` was rejected: it has no stealth patches and needs a Node runtime in the
  worker image. go-rod's launcher kills Chrome when the worker exits, so `Chrome` has no `Close`.
- **Chrome goes out through `proxy.Tunnel`** (#1020), a local CONNECT proxy on `127.0.0.1` that adds the
  Decodo credentials and a sticky session. Chrome cannot take proxy credentials on its command line, and
  the Tunnel gives it one exit IP until `Tunnel.Rotate`. Chrome keeps CONNECT connections alive, so a
  rotation also relaunches the browser.
- **Only the first page and recovery navigate.** `/classic?searchState=…` is loaded until Cloudflare clears
  and `script#__NEXT_DATA__` yields `buildId` and page 0's `pageProps`. Later pages and descriptions are
  `fetch()` calls from inside the cleared page. Images, fonts and media are blocked to save proxy bytes.
- **Recovery ladder** on a challenge or stale `buildId`: reload; then `Tunnel.Rotate` and reload; then
  `gate.Block` and return the gate's `*sources.DeferredError`.
- **`locations` is always sent** in the search state; otherwise hiring.cafe defaults the region to the exit
  IP's country.

## Personal use only

hiring.cafe's terms forbid republishing its listings, and `robots.txt` disallows `?searchState=` and
`?page=`. This Source exists to read one user's own searches. **Offering it to other users needs a rethink**
(permission from hiring.cafe, or dropping the Source) before anything ships.

## Spike outcome

`just cli hiringcafe <search-url>` runs the tab for about an hour over a search's pages and descriptions and
prints pages fetched, descriptions fetched, challenges met, rotations and Decodo bytes.

TODO(human, #1024): run it with `DECODO_PROXY_URL` set and record here:

| Measure | Result |
|---|---|
| Pages fetched | TODO |
| Descriptions fetched | TODO |
| Challenges met | TODO |
| Rotations | TODO |
| Decodo bytes | TODO |
| Go or no-go | TODO |

If no-go, the fallback is a userscript that posts to `/ingest`, filed as a new issue.

## Consequences

- The worker image needs Chrome (or go-rod downloads one on first launch).
- Every hiring.cafe request costs proxy bytes and a ~3s gate gap.
- A Cloudflare change can end the Source; the Gate block keeps a failing run from hammering the host.

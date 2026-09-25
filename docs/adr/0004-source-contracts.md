# ADR 0004 — Source contracts: URL classification and optional capabilities

- **Classify before routing.** `detect.Detect(url)` is a pure function that labels a URL as a known ATS, an Aggregator, or unknown HTML. Aggregator URLs are rewritten to their underlying ATS where possible, so the job is fetched from the cheap ATS API rather than the hostile aggregator. Adding an ATS means adding a `Detect` case plus a source, and table tests keep the two in step.
- **Capabilities are types, not flags.** Every source fetches pages. Fetching details is a separate, optional interface that only HTML/discovery sources implement. ATS sources return complete jobs from the listing and carry no stub methods.
- **Fetching is a dependency, not a source variant.** Sources parse and paginate; how bytes are fetched (direct or Web Unlocker, see ADR 0009) is injected, so no parser knows about proxies.

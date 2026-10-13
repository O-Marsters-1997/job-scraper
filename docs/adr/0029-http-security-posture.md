# ADR 0029 — HTTP security posture

Issue: [#874](https://github.com/O-Marsters-1997/job-scraper/issues/874). Records what W1 (#925) built.
The go-idiomatic skill (`middleware.md`, `secrets.md`) is the authority.

## Decision

- **Same site, so SameSite=Lax.** The frontend and the API share a site. Cookies are always `SameSite=Lax`
  (#906). `http.CrossOriginProtection` checks `Origin` against the CORS origin on unsafe methods.
- **Bodies are capped and typed.** A 1 MiB request body cap, and JSON `Content-Type` is enforced in middleware
  (#877). Handlers never check either.
- **Session tokens are stored as SHA-256 hashes** (#912). Deploying it logged every user out once.
- **Auth is rate limited in memory**, per IP and per username (#915). The limiter is per process and resets
  on restart, which is acceptable at one API replica.
- **Login is constant time**, with one bcrypt cost for every outcome, including an unknown username (#915).
- **A password reset revokes all of the user's sessions** (#879, shipped inside #912).
- **tokencrypt v1** (#916): a version byte plus AAD of table and owner id, so a ciphertext cannot be moved to
  another row or user. v0 stays readable and is rewritten on the next write. `admin reencrypt` sweeps the rest.
  Deleting v0 support is a follow-up once the sweep has run in production.
- **Servers shut down gracefully** (#902) and set read, write and idle timeouts (#905).

## Consequences

- The in-memory limiter does not survive a restart or a second replica; moving to Postgres or Redis is a
  decision for when the API scales out.
- Until v0 is deleted, tokencrypt carries two read paths.

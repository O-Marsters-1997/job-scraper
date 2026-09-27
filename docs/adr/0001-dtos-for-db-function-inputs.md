# ADR 0001 — DTOs for DB function inputs

When a store method takes more than a handful of scalar arguments, its input struct is named for what it carries (`CreateApplicationInput`), never `…Params`. It lives in `dto` when it is also an HTTP body or crosses a context facade. Otherwise it lives in the owning context's store package (amended by ADR 0011, which narrows `dto` to wire and cross-context shapes and removes `providers`).

# ADR 0007 — Store salary raw, defer normalisation

Sources report salary inconsistently (ranges, currencies, hourly or annual, "competitive"). `jobs.salary_raw TEXT` stores exactly what the source gave. It is a property of the shared Job, and the LLM rubric reads it as text. Numeric min/max columns and salary filtering are deferred: a parser would be ongoing per-source maintenance, and LLM normalisation would tie accuracy to model output. Adding them later is additive (new nullable columns plus a backfill).

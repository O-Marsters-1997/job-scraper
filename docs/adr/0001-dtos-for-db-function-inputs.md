# ADR 0001 — DTOs for DB function inputs

When a db method takes more than a handful of scalar arguments, its input struct lives in `dto` and is named for what it carries (`CreateApplicationInput`), not in `providers` as `…Params`. `dto` already owns every data shape passed between layers. `providers` holds only interfaces and sentinel errors.

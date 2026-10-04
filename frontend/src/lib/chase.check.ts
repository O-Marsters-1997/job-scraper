import assert from "node:assert/strict";
import { chaseDateKey, formatChaseDate, isChaseOverdue } from "./chase";

const now = new Date(2026, 9, 20, 15, 30);

assert.equal(chaseDateKey("2026-10-20T00:00:00Z"), "2026-10-20");
assert.equal(isChaseOverdue("2026-10-19T00:00:00Z", now), true);
assert.equal(isChaseOverdue("2026-10-20T00:00:00Z", now), false);
assert.equal(isChaseOverdue("2026-10-21T00:00:00Z", now), false);
assert.equal(formatChaseDate("2026-10-20T00:00:00Z"), "20 Oct 2026");

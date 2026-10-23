import assert from "node:assert/strict";
import {
	formatDuration,
	newerPage,
	olderPage,
	parseActivitySearch,
} from "./scrapeRuns";

const second = olderPage({}, "c1");
assert.deepEqual(second, { before: "c1", trail: undefined });
const third = olderPage(second, "c2");
assert.deepEqual(third, { before: "c2", trail: "c1" });
assert.deepEqual(newerPage(third), { before: "c1", trail: undefined });
assert.deepEqual(newerPage(second), { before: undefined, trail: undefined });

assert.equal(parseActivitySearch({ trail: "c1" }).trail, undefined);
assert.equal(parseActivitySearch({ before: "c2", trail: "c1" }).trail, "c1");

assert.equal(
	formatDuration("2026-10-23T10:00:00Z", "2026-10-23T10:00:15Z"),
	"15s",
);
assert.equal(
	formatDuration("2026-10-23T10:00:00Z", "2026-10-23T10:02:05Z"),
	"2m 5s",
);

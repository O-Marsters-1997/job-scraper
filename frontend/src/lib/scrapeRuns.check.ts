import assert from "node:assert/strict";
import type { SourceHealth } from "@/types/scrapeRun";
import {
	activeFilterCount,
	clearFilters,
	customRangeError,
	formatDuration,
	healthLabel,
	newerPage,
	olderPage,
	onlyFacetValue,
	pageSummary,
	parseActivitySearch,
	setRange,
	splitHealth,
	toggleFacetValue,
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

const health = (
	Status: SourceHealth["Status"],
	LastRunAt: string | null,
): SourceHealth => ({
	Kind: "search",
	TargetID: "t",
	Source: "indeed",
	TargetValue: "go",
	TargetFilters: {},
	Status,
	Reason: "ok",
	Recent: [],
	LastRunAt,
	New: 0,
	Elsewhere: 0,
	Retries: 2,
});
const split = splitHealth([
	health("green", "2026-10-23T10:00:00Z"),
	health("amber", "2026-10-23T10:00:00Z"),
	health("red", "2026-10-23T09:00:00Z"),
	health("red", "2026-10-23T11:00:00Z"),
]);
assert.deepEqual(
	split.attention.map((r) => `${r.Status}@${r.LastRunAt}`),
	[
		"red@2026-10-23T11:00:00Z",
		"red@2026-10-23T09:00:00Z",
		"amber@2026-10-23T10:00:00Z",
	],
);
assert.equal(split.healthy.length, 1);
assert.equal(
	healthLabel({ ...health("amber", null), Reason: "retried", Retries: 1 }),
	"1 retried attempt",
);

const paged = { before: "c2", trail: "c1" };
const failed = toggleFacetValue(paged, "outcome", "failed");
assert.deepEqual(failed, {
	before: undefined,
	trail: undefined,
	outcome: "failed",
});
const widened = toggleFacetValue(failed, "outcome", "retried");
assert.equal(widened.outcome, "failed,retried");
assert.equal(toggleFacetValue(widened, "outcome", "failed").outcome, "retried");
assert.equal(toggleFacetValue(failed, "outcome", "failed").outcome, undefined);
assert.equal(onlyFacetValue(widened, "outcome", "retried").outcome, "retried");
assert.equal(
	activeFilterCount({ outcome: "failed,retried", source: "linkedin" }),
	3,
);
assert.equal(activeFilterCount(clearFilters({ outcome: "failed", q: "x" })), 0);
assert.equal(setRange({ range: "1h" }, "7d").range, undefined);

assert.deepEqual(
	parseActivitySearch({ range: "2d", from: "a", to: "b" }),
	parseActivitySearch({ from: "a", to: "b" }),
);
assert.equal(parseActivitySearch({ range: "2d" }).range, undefined);
assert.equal(parseActivitySearch({ range: "6h", from: "a" }).range, "6h");

assert.equal(pageSummary({}, 50, 120), "1–50 of 120");
assert.equal(pageSummary({ before: "c1" }, 50, 120), "51–100 of 120");
assert.equal(
	pageSummary({ before: "c2", trail: "c1" }, 20, 120),
	"101–120 of 120",
);
assert.equal(pageSummary({}, 0, 0), "0 of 0");

assert.equal(
	customRangeError("2026-10-01T00:00:00Z", "2026-10-02T00:00:00Z"),
	null,
);
assert.ok(customRangeError("2026-10-02T00:00:00Z", "2026-10-01T00:00:00Z"));
assert.ok(customRangeError("2026-09-01T00:00:00Z", "2026-10-02T00:00:00Z"));
assert.ok(customRangeError("", "2026-10-02T00:00:00Z"));

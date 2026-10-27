import assert from "node:assert/strict";
import type { ScrapeRun, SourceHealth } from "@/types/scrapeRun";
import {
	activeFilterCount,
	byNew24h,
	clearFilters,
	customRangeError,
	formatDuration,
	groupPollsByDay,
	healthLabel,
	newerPage,
	olderPage,
	onlyFacetValue,
	pageSummary,
	parseActivitySearch,
	parseHistorySearch,
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
	BoardID: "",
	CompanyName: "",
	Status,
	Reason: "ok",
	Recent: [],
	LastRunAt,
	New: 0,
	New24h: 0,
	Elsewhere: 0,
	Removed: 0,
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
const withNew = [
	{ ...health("green", null), New24h: 1 },
	{ ...health("red", null), New24h: 0 },
	{ ...health("amber", null), New24h: 5 },
];
assert.deepEqual(
	byNew24h(withNew, { onlyNew: false, sortByNew: true }).map((r) => r.New24h),
	[5, 1, 0],
);
assert.deepEqual(
	byNew24h(withNew, { onlyNew: true, sortByNew: false }).map((r) => r.New24h),
	[1, 5],
);
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

const poll = (
	ID: string,
	StartedAt: string,
	over: Partial<ScrapeRun> = {},
): ScrapeRun => ({
	ID,
	Kind: "board",
	Source: "greenhouse",
	TargetID: "",
	TargetValue: "",
	TargetFilters: {},
	BoardID: "b",
	CompanyName: "Co",
	Trigger: "scheduled",
	Outcome: "succeeded",
	StopReason: "last_page",
	StartedAt,
	FinishedAt: StartedAt,
	Pages: 0,
	Listings: 8,
	New: 0,
	Elsewhere: 0,
	Repeat: 0,
	Removed: 0,
	Reopened: 0,
	Deferred: 0,
	Retries: 0,
	Errors: [],
	Changes: [],
	...over,
});

const days = groupPollsByDay([
	poll("a", "2026-10-23T12:00:00", { New: 1 }),
	poll("b", "2026-10-23T11:00:00"),
	poll("c", "2026-10-23T10:00:00"),
	poll("d", "2026-10-23T09:00:00", { Outcome: "failed" }),
	poll("e", "2026-10-22T09:00:00"),
	poll("f", "2026-10-22T08:00:00", { Reopened: 1 }),
]);
assert.deepEqual(
	days.map((d) => d.entries.map((e) => e.kind)),
	[
		["poll", "quiet", "poll"],
		["quiet", "poll"],
	],
);
const quiet = days[0]?.entries[1];
assert.equal(quiet?.kind === "quiet" && quiet.runs.length, 2);

assert.equal(parseHistorySearch({ tab: "history" }).tab, "history");
assert.equal(parseHistorySearch({ tab: "bogus" }).tab, undefined);

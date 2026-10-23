import assert from "node:assert/strict";
import type { SourceHealth } from "@/types/scrapeRun";
import {
	formatDuration,
	healthLabel,
	newerPage,
	olderPage,
	parseActivitySearch,
	splitHealth,
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

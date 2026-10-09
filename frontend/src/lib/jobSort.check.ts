import assert from "node:assert/strict";
import { best, newest, sortJobs } from "./jobSort";

const job = (ID: string, score: number | null, discovered: string) =>
	({
		ID,
		SuitabilityScore: score,
		FirstDiscoveredAt: discovered,
		ScrapedAt: "2020-01-01T00:00:00Z",
	}) as never;

const jobs = [
	job("old-high", 90, "2026-01-01T00:00:00Z"),
	job("new-unscored", null, "2026-03-01T00:00:00Z"),
	job("mid-high", 90, "2026-02-01T00:00:00Z"),
	job("new-low", 40, "2026-03-02T00:00:00Z"),
	job("old-unscored", null, "2026-01-02T00:00:00Z"),
] as { ID: string }[] as never[];

const ids = (list: { ID: string }[]) => list.map((j) => j.ID);

assert.deepEqual(ids(newest(jobs)), [
	"new-low",
	"new-unscored",
	"mid-high",
	"old-unscored",
	"old-high",
]);

assert.deepEqual(ids(best(jobs)), [
	"mid-high",
	"old-high",
	"new-low",
	"new-unscored",
	"old-unscored",
]);

assert.equal(sortJobs(jobs, "relevant"), jobs);
assert.deepEqual(ids(sortJobs(jobs, "best")), ids(best(jobs)));
assert.deepEqual(ids(sortJobs(jobs, "newest")), ids(newest(jobs)));

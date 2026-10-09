import assert from "node:assert/strict";
import { best, newest, relevanceRank, relevant, sortJobs } from "./jobSort";

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

const now = Date.parse("2026-03-10T00:00:00Z");
const day = 86_400_000;
const ago = (days: number) => new Date(now - days * day).toISOString();

assert.ok(
	Math.abs(
		relevanceRank(job("a", 80, ago(3)), now) -
			relevanceRank(job("b", 40, ago(0)), now),
	) < 1e-9,
);
assert.ok(
	relevanceRank(job("a", 90, ago(0)), now) >
		relevanceRank(job("b", 60, ago(0)), now),
);
assert.equal(
	relevanceRank(job("a", null, ago(0)), now),
	relevanceRank(job("b", 50, ago(0)), now),
);
assert.equal(
	relevanceRank(job("a", 70, ago(-5)), now),
	relevanceRank(job("b", 70, ago(0)), now),
);

const tied = [
	job("b", 50, ago(1)),
	job("a", 50, ago(1)),
	job("fresh", 25, ago(-1)),
	job("c", 50, ago(0)),
];
assert.deepEqual(ids(relevant(tied, now)), ["c", "a", "b", "fresh"]);
assert.deepEqual(
	ids(sortJobs(tied, "relevant", now)),
	ids(relevant(tied, now)),
);
assert.deepEqual(ids(sortJobs(jobs, "best")), ids(best(jobs)));
assert.deepEqual(ids(sortJobs(jobs, "newest")), ids(newest(jobs)));

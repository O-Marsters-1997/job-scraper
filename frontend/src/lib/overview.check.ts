import assert from "node:assert/strict";
import {
	applicationStats,
	chasesDue,
	highValueJobs,
	jobStats,
	pipelineSegments,
} from "./overview";

const now = new Date("2026-06-10T12:00:00");
const job = (id: string, source: string, scrapedAt: string) =>
	({ ID: id, Source: source, ScrapedAt: scrapedAt }) as never;
const app = (statusID: string) => ({ StatusID: statusID }) as never;
const status = (id: string) => ({ ID: id, Name: id, Colour: "#000" }) as never;

const jobs = [
	job("a", "lever", "2026-06-10T08:00:00"),
	job("b", "lever", "2026-06-10T09:00:00"),
	job("c", "ashby", "2026-06-09T09:00:00"),
];
assert.deepEqual(jobStats(jobs, now), {
	total: 3,
	sources: 2,
	newToday: 2,
	todaySources: 1,
});
assert.deepEqual(jobStats([], now), {
	total: 0,
	sources: 0,
	newToday: 0,
	todaySources: 0,
});

const statuses = [status("applied"), status("screen"), status("offer")];
const apps = [app("applied"), app("screen"), app("offer"), app("offer")];
assert.deepEqual(applicationStats(apps, statuses), {
	total: 4,
	awaiting: 2,
	responded: 2,
	responseRate: 50,
});
assert.deepEqual(applicationStats([], statuses), {
	total: 0,
	awaiting: 0,
	responded: 0,
	responseRate: 0,
});

assert.deepEqual(
	pipelineSegments(apps, statuses).map((s) => s.count),
	[1, 1, 2],
);
assert.deepEqual(pipelineSegments(apps, []), []);

const hv = (id: string, over: Record<string, unknown>) =>
	({
		ID: id,
		ScrapedAt: "2026-06-09T09:00:00",
		SuitabilityScore: 80,
		Band: "good",
		Seen: false,
		...over,
	}) as never;
const ids = (r: { jobs: { ID: string }[] }) => r.jobs.map((j) => j.ID);

const ranked = highValueJobs(
	[
		hv("low", { SuitabilityScore: 70 }),
		hv("old", { ScrapedAt: "2026-05-30T09:00:00" }),
		hv("seen", { Seen: true }),
		hv("fair", { Band: "fair" }),
		hv("newer", { ScrapedAt: "2026-06-10T09:00:00" }),
		hv("tie", {}),
		hv("top", { SuitabilityScore: 95, Band: "great" }),
		hv("unscored", { SuitabilityScore: null, Band: "" }),
	],
	true,
	now,
);

assert.deepEqual(ids(ranked), ["top", "newer", "tie", "low"]);
assert.equal(
	highValueJobs(
		Array.from({ length: 9 }, (_, i) => hv(`j${i}`, {})),
		true,
		now,
	).jobs.length,
	5,
);

const none = { SuitabilityScore: null, Band: "" };
const unranked = highValueJobs(
	[
		hv("a", { ...none, ScrapedAt: "2026-05-01T09:00:00" }),
		hv("b", { ...none, ScrapedAt: "2026-06-01T09:00:00" }),
		hv("c", { ...none, Seen: true }),
	],
	false,
	now,
);
assert.equal(unranked.ranked, false);
assert.deepEqual(ids(unranked), ["b", "a"]);

const chaseApp = (id: string, chaseBy: string | null) =>
	({ ID: id, ChaseBy: chaseBy }) as never;
const chases = [
	chaseApp("tomorrow", "2026-06-11T00:00:00Z"),
	chaseApp("today", "2026-06-10T00:00:00Z"),
	chaseApp("none", null),
	chaseApp("yesterday", "2026-06-09T00:00:00Z"),
];
assert.deepEqual(
	chasesDue(chases, now).map((a) => a.ID),
	["yesterday", "today"],
);
assert.deepEqual(chasesDue([], now), []);

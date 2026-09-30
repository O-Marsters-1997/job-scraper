import assert from "node:assert/strict";
import {
	applicationStats,
	jobStats,
	pipelineSegments,
	recentJobs,
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

assert.deepEqual(
	recentJobs(jobs, 2).map((j) => j.ID),
	["b", "a"],
);
assert.ok(recentJobs(jobs, 10).length === 3);

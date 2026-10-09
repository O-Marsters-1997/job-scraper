import type { ApplicationWithDetails } from "@/types/application";
import type { ApplicationStatus } from "@/types/applicationStatus";
import type { Job } from "@/types/job";
import { chaseDateKey } from "./chase";
import { dayKey } from "./datetime";

export const RECENT_LIMIT = 5;

export function jobStats(jobs: Job[], now = new Date()) {
	const today = jobs.filter((j) => dayKey(j.ScrapedAt) === dayKey(now));
	return {
		total: jobs.length,
		sources: new Set(jobs.map((j) => j.Source)).size,
		newToday: today.length,
		todaySources: new Set(today.map((j) => j.Source)).size,
	};
}

export function applicationStats(
	applications: ApplicationWithDetails[],
	statuses: ApplicationStatus[],
) {
	const early = new Set(statuses.slice(0, 2).map((s) => s.ID));
	const total = applications.length;
	const awaiting = applications.filter((a) => early.has(a.StatusID)).length;
	const responded = total - awaiting;
	return {
		total,
		awaiting,
		responded,
		responseRate: total > 0 ? Math.round((responded / total) * 100) : 0,
	};
}

export function pipelineSegments(
	applications: ApplicationWithDetails[],
	statuses: ApplicationStatus[],
) {
	const counts = new Map<string, number>();
	for (const app of applications) {
		counts.set(app.StatusID, (counts.get(app.StatusID) ?? 0) + 1);
	}
	return statuses.map((s) => ({ ...s, count: counts.get(s.ID) ?? 0 }));
}

const HIGH_VALUE_DAYS = 7;
const DAY_MS = 24 * 60 * 60 * 1000;

function scrapedMs(job: Job) {
	return new Date(job.ScrapedAt).getTime();
}

export function highValueJobs(
	jobs: Job[],
	ranked: boolean,
	now = new Date(),
	limit = RECENT_LIMIT,
) {
	const cutoff = now.getTime() - HIGH_VALUE_DAYS * DAY_MS;
	const picked = ranked
		? jobs
				.filter(
					(j) =>
						!j.Seen &&
						scrapedMs(j) >= cutoff &&
						(j.Band === "great" || j.Band === "good"),
				)
				.sort(
					(a, b) =>
						(b.SuitabilityScore ?? 0) - (a.SuitabilityScore ?? 0) ||
						scrapedMs(b) - scrapedMs(a),
				)
		: jobs.filter((j) => !j.Seen).sort((a, b) => scrapedMs(b) - scrapedMs(a));
	return { ranked, jobs: picked.slice(0, limit) };
}

export function chasesDue(
	applications: ApplicationWithDetails[],
	now = new Date(),
) {
	const today = dayKey(now);
	return applications
		.filter((a) => a.ChaseBy != null && chaseDateKey(a.ChaseBy) <= today)
		.sort((a, b) =>
			chaseDateKey(a.ChaseBy!).localeCompare(chaseDateKey(b.ChaseBy!)),
		);
}

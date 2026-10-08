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

export function recentJobs(jobs: Job[], limit = RECENT_LIMIT) {
	return [...jobs]
		.sort(
			(a, b) =>
				new Date(b.ScrapedAt).getTime() - new Date(a.ScrapedAt).getTime(),
		)
		.slice(0, limit);
}

export function chasesDue(
	applications: ApplicationWithDetails[],
	now = new Date(),
) {
	const today = dayKey(now);
	return applications
		.filter((a) => a.ChaseBy !== null && chaseDateKey(a.ChaseBy) <= today)
		.sort((a, b) =>
			chaseDateKey(a.ChaseBy!).localeCompare(chaseDateKey(b.ChaseBy!)),
		);
}

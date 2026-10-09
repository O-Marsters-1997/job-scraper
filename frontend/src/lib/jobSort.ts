import type { Job } from "@/types/job";

export const SORT_OPTIONS = ["relevant", "newest", "best"] as const;
export type JobSort = (typeof SORT_OPTIONS)[number];

export const SORT_LABELS: Record<JobSort | "custom", string> = {
	relevant: "Relevant",
	newest: "Newest",
	best: "Best match",
	custom: "Custom",
};

const discoveredAt = (job: Job) =>
	new Date(job.FirstDiscoveredAt ?? job.ScrapedAt).getTime();

const byNewest = (a: Job, b: Job) => discoveredAt(b) - discoveredAt(a);

export function newest<T extends Job>(jobs: T[]): T[] {
	return [...jobs].sort(byNewest);
}

export function best<T extends Job>(jobs: T[]): T[] {
	return [...jobs].sort((a, b) => {
		if (a.SuitabilityScore == null && b.SuitabilityScore == null)
			return byNewest(a, b);
		if (a.SuitabilityScore == null) return 1;
		if (b.SuitabilityScore == null) return -1;
		return b.SuitabilityScore - a.SuitabilityScore || byNewest(a, b);
	});
}

export const HALF_LIFE_DAYS = 3;
export const UNSCORED_PRIOR = 50;
const MS_PER_DAY = 86_400_000;

export function relevanceRank(job: Job, now: number = Date.now()): number {
	const ageDays = Math.max(0, (now - discoveredAt(job)) / MS_PER_DAY) || 0;
	const score = job.SuitabilityScore ?? UNSCORED_PRIOR;
	return score * 0.5 ** (ageDays / HALF_LIFE_DAYS);
}

export function relevant<T extends Job>(
	jobs: T[],
	now: number = Date.now(),
): T[] {
	return jobs
		.map((job) => ({
			job,
			rank: relevanceRank(job, now),
			at: discoveredAt(job) || 0,
		}))
		.sort(
			(a, b) =>
				b.rank - a.rank || b.at - a.at || a.job.ID.localeCompare(b.job.ID),
		)
		.map(({ job }) => job);
}

export function sortJobs<T extends Job>(
	jobs: T[],
	sort: JobSort,
	now: number = Date.now(),
): T[] {
	switch (sort) {
		case "newest":
			return newest(jobs);
		case "best":
			return best(jobs);
		case "relevant":
			return relevant(jobs, now);
	}
}

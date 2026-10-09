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

export function sortJobs<T extends Job>(jobs: T[], sort: JobSort): T[] {
	switch (sort) {
		case "newest":
			return newest(jobs);
		case "best":
			return best(jobs);
		case "relevant":
			return jobs;
	}
}

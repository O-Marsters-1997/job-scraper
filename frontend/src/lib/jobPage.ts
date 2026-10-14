import type { Job } from "@/types/job";

export interface JobPage {
	items: Job[];
	total: number;
}

export function patchJobs(
	page: JobPage,
	matches: (job: Job) => boolean,
	patch: Partial<Job>,
): JobPage {
	return {
		...page,
		items: page.items.map((job) => (matches(job) ? { ...job, ...patch } : job)),
	};
}

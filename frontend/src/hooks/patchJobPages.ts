import type { QueryClient } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import { type JobPage, patchJobs } from "../lib/jobPage";
import type { Job } from "../types/job";

export function patchJobPages(
	queryClient: QueryClient,
	matches: (job: Job) => boolean,
	patch: Partial<Job>,
) {
	queryClient.setQueriesData<JobPage>(
		{ queryKey: keys.jobs.page() },
		(page) => page && patchJobs(page, matches, patch),
	);
}

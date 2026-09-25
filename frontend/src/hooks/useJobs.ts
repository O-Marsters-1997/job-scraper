import { createQuery, queryOptions } from "@tanstack/solid-query";
import { fetchAllJobs, fetchJob, fetchJobs } from "../api/jobs";

export const jobsQueryOptions = queryOptions({
	queryKey: ["jobs"],
	queryFn: () => fetchJobs().then((page) => page.items),
});

export const allJobsQueryOptions = queryOptions({
	queryKey: ["jobs", "all"],
	queryFn: fetchAllJobs,
});

export function useAllJobs() {
	return createQuery(() => allJobsQueryOptions);
}

export function jobQueryOptions(id: string) {
	return queryOptions({ queryKey: ["job", id], queryFn: () => fetchJob(id) });
}

export function useJob(id: () => string) {
	return createQuery(() => jobQueryOptions(id()));
}

export function useJobs() {
	return createQuery(() => jobsQueryOptions);
}

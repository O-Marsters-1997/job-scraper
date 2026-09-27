import { createQuery, queryOptions } from "@tanstack/solid-query";
import { fetchAllJobs, fetchJob } from "../api/jobs";

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

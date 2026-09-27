import { createQuery, queryOptions } from "@tanstack/solid-query";
import { fetchAllJobs, fetchJob } from "../api/jobs";
import { keys } from "../api/keys";

export const allJobsQueryOptions = queryOptions({
	queryKey: keys.jobs.full(),
	queryFn: fetchAllJobs,
});

export function useAllJobs() {
	return createQuery(() => allJobsQueryOptions);
}

export function jobQueryOptions(id: string) {
	return queryOptions({
		queryKey: keys.jobs.detail(id),
		queryFn: () => fetchJob(id),
	});
}

export function useJob(id: () => string) {
	return createQuery(() => jobQueryOptions(id()));
}

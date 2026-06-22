import { createQuery, queryOptions } from "@tanstack/solid-query";
import { fetchJobs } from "../api/jobs";

export const jobsQueryOptions = queryOptions({
	queryKey: ["jobs"],
	queryFn: fetchJobs,
});

export function useJobs() {
	return createQuery(() => jobsQueryOptions);
}

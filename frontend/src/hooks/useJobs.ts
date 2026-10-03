import { createQuery, queryOptions } from "@tanstack/solid-query";
import { fetchAllJobs, fetchJob, markJobsSeen } from "../api/jobs";
import { keys } from "../api/keys";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

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

export function useMarkJobsSeen() {
	return useInvalidatingMutation(markJobsSeen, [keys.jobs.all]);
}

export function useMarkSeenAfterGrade() {
	const markSeen = useMarkJobsSeen();
	return async (jobIds: string[]) => {
		if (jobIds.length === 0) return;
		await markSeen.mutateAsync({ jobIds, seen: true }).catch(() => undefined);
	};
}

import {
	createMutation,
	createQuery,
	queryOptions,
	useQueryClient,
} from "@tanstack/solid-query";
import { fetchJobs, requestJobReasoning } from "../api/jobs";

export const jobsQueryOptions = queryOptions({
	queryKey: ["jobs"],
	queryFn: fetchJobs,
});

export function useJobs() {
	return createQuery(() => jobsQueryOptions);
}

export function useRequestReasoning() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: (jobId: string) => requestJobReasoning(jobId),
		onSuccess: () => queryClient.invalidateQueries({ queryKey: ["jobs"] }),
	}));
}

import {
	createQuery,
	queryOptions,
	useQueryClient,
} from "@tanstack/solid-query";
import { createMemo } from "solid-js";
import {
	fetchJob,
	fetchJobInsights,
	fetchJobOverview,
	fetchJobPage,
	markJobsSeen,
} from "../api/jobs";
import { keys } from "../api/keys";
import type { JobFilters } from "../lib/jobFilters";
import type { JobPage } from "../lib/jobPage";
import type { Job } from "../types/job";
import { patchJobPages } from "./patchJobPages";
import { useInvalidatingMutation } from "./useInvalidatingMutation";
import { useServerTable } from "./useServerTable";

const JOBS_PAGE_SIZE = 25;

export const jobInsightsQueryOptions = queryOptions({
	queryKey: keys.jobs.insights(),
	queryFn: fetchJobInsights,
});

export const jobOverviewQueryOptions = (ranked: boolean) =>
	queryOptions({
		queryKey: keys.jobs.overview(ranked),
		queryFn: () => fetchJobOverview(ranked),
	});

export function useJobOverview(ranked: () => boolean) {
	return createQuery(() => jobOverviewQueryOptions(ranked()));
}

export function useJobInsights() {
	return createQuery(() => jobInsightsQueryOptions);
}

export function useJobPages(
	state: () => JobFilters,
	onChange: (patch: Partial<JobFilters>) => void,
) {
	const queryClient = useQueryClient();
	return useServerTable({
		state,
		onChange,
		pageSize: JOBS_PAGE_SIZE,
		query: (filters) => ({
			queryKey: keys.jobs.list(filters),
			queryFn: () => {
				const firstPage = queryClient.getQueryData<JobPage>(
					keys.jobs.list({ ...filters, page: undefined }),
				);
				const knownTotal = filters.page ? firstPage?.total : undefined;
				return fetchJobPage(filters, JOBS_PAGE_SIZE, knownTotal);
			},
		}),
		rowId: (job) => job.ID,
	});
}

export function jobQueryOptions(id: string) {
	return queryOptions({
		queryKey: keys.jobs.detail(id),
		queryFn: () => fetchJob(id),
	});
}

export function useJob(id: () => string) {
	const stableId = createMemo(id);
	return createQuery(() => jobQueryOptions(stableId()));
}

export function useMarkJobsSeen() {
	return useInvalidatingMutation(markJobsSeen, [keys.jobs.page()], {
		patch: (queryClient, _result, { jobIds, seen }) => {
			const ids = new Set(jobIds);
			patchJobPages(queryClient, (job) => ids.has(job.ID), { Seen: seen });
			for (const id of jobIds) {
				queryClient.setQueryData<Job>(
					keys.jobs.detail(id),
					(job) => job && { ...job, Seen: seen },
				);
			}
		},
	});
}

export function useMarkSeenAfterGrade() {
	const markSeen = useMarkJobsSeen();
	return async (jobIds: string[]) => {
		if (jobIds.length === 0) return;
		await markSeen.mutateAsync({ jobIds, seen: true }).catch(() => undefined);
	};
}

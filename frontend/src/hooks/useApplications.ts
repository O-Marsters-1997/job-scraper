import {
	createMutation,
	createQuery,
	queryOptions,
	useQueryClient,
} from "@tanstack/solid-query";
import {
	createApplication,
	deleteApplication,
	fetchApplications,
	fetchApplicationsForJobs,
	updateApplication,
} from "../api/applications";

export const applicationsQueryOptions = (statusId?: string) =>
	queryOptions({
		queryKey: ["applications", statusId ?? "all"],
		queryFn: () => fetchApplications(statusId),
	});

export function useApplications(statusId?: () => string | undefined) {
	return createQuery(() => applicationsQueryOptions(statusId?.()));
}

export const applicationsForJobsQueryOptions = (jobIds: string[]) =>
	queryOptions({
		queryKey: ["applications-for-jobs", jobIds],
		queryFn: () => fetchApplicationsForJobs(jobIds),
		enabled: jobIds.length > 0,
	});

export function useApplicationsForJobs(jobIds: () => string[]) {
	return createQuery(() => applicationsForJobsQueryOptions(jobIds()));
}

export function useCreateApplication() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: createApplication,
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["applications"] });
			queryClient.invalidateQueries({ queryKey: ["applications-for-jobs"] });
		},
	}));
}

export function useUpdateApplication() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: ({
			id,
			data,
		}: {
			id: string;
			data: Parameters<typeof updateApplication>[1];
		}) => updateApplication(id, data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["applications"] });
			queryClient.invalidateQueries({ queryKey: ["applications-for-jobs"] });
		},
	}));
}

export function useDeleteApplication() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: deleteApplication,
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["applications"] });
			queryClient.invalidateQueries({ queryKey: ["applications-for-jobs"] });
		},
	}));
}

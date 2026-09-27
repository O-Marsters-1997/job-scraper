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
	updateApplication,
} from "../api/applications";
import { keys } from "../api/keys";

export const applicationsQueryOptions = (statusId?: string) =>
	queryOptions({
		queryKey: keys.applications.byStatus(statusId),
		queryFn: () => fetchApplications(statusId),
	});

export function useApplications(statusId?: () => string | undefined) {
	return createQuery(() => applicationsQueryOptions(statusId?.()));
}

export function useCreateApplication() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: createApplication,
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: keys.applications.all });
			queryClient.invalidateQueries({ queryKey: keys.jobs.all });
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
			queryClient.invalidateQueries({ queryKey: keys.applications.all });
			queryClient.invalidateQueries({ queryKey: keys.jobs.all });
		},
	}));
}

export function useDeleteApplication() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: deleteApplication,
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: keys.applications.all });
			queryClient.invalidateQueries({ queryKey: keys.jobs.all });
		},
	}));
}

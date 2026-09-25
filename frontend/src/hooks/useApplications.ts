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

export const applicationsQueryOptions = (statusId?: string) =>
	queryOptions({
		queryKey: ["applications", statusId ?? "all"],
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
			queryClient.invalidateQueries({ queryKey: ["applications"] });
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
		},
	}));
}

export function useDeleteApplication() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: deleteApplication,
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["applications"] });
		},
	}));
}

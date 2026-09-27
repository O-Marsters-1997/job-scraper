import {
	createMutation,
	createQuery,
	queryOptions,
	useQueryClient,
} from "@tanstack/solid-query";
import {
	createApplicationStatus,
	deleteApplicationStatus,
	fetchApplicationStatuses,
	updateApplicationStatus,
} from "../api/applicationStatuses";
import { keys } from "../api/keys";

export const applicationStatusesQueryOptions = queryOptions({
	queryKey: keys.statuses,
	queryFn: fetchApplicationStatuses,
});

export function useApplicationStatuses() {
	return createQuery(() => applicationStatusesQueryOptions);
}

export function useCreateApplicationStatus() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: ({ name, colour }: { name: string; colour: string }) =>
			createApplicationStatus(name, colour),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: keys.statuses });
			queryClient.invalidateQueries({ queryKey: keys.applications.all });
		},
	}));
}

export function useUpdateApplicationStatus() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: ({
			id,
			name,
			colour,
		}: {
			id: string;
			name: string;
			colour: string;
		}) => updateApplicationStatus(id, name, colour),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: keys.statuses });
			queryClient.invalidateQueries({ queryKey: keys.applications.all });
		},
	}));
}

export function useDeleteApplicationStatus() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: (id: string) => deleteApplicationStatus(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: keys.statuses });
			queryClient.invalidateQueries({ queryKey: keys.applications.all });
		},
	}));
}

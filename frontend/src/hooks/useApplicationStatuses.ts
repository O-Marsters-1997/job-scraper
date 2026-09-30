import { createQuery, queryOptions } from "@tanstack/solid-query";
import {
	createApplicationStatus,
	deleteApplicationStatus,
	fetchApplicationStatuses,
	updateApplicationStatus,
} from "../api/applicationStatuses";
import { keys } from "../api/keys";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export const applicationStatusesQueryOptions = queryOptions({
	queryKey: keys.statuses,
	queryFn: fetchApplicationStatuses,
});

export function useApplicationStatuses() {
	return createQuery(() => applicationStatusesQueryOptions);
}

export function useCreateApplicationStatus() {
	return useInvalidatingMutation(
		({ name, colour }: { name: string; colour: string }) =>
			createApplicationStatus(name, colour),
		[keys.statuses, keys.applications.all],
	);
}

export function useUpdateApplicationStatus() {
	return useInvalidatingMutation(
		({ id, name, colour }: { id: string; name: string; colour: string }) =>
			updateApplicationStatus(id, name, colour),
		[keys.statuses, keys.applications.all],
	);
}

export function useDeleteApplicationStatus() {
	return useInvalidatingMutation(deleteApplicationStatus, [
		keys.statuses,
		keys.applications.all,
	]);
}

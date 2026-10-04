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
		({
			name,
			colour,
			replyWindowDays,
		}: {
			name: string;
			colour: string;
			replyWindowDays: number | null;
		}) => createApplicationStatus(name, colour, replyWindowDays),
		[keys.statuses, keys.applications.all],
	);
}

export function useUpdateApplicationStatus() {
	return useInvalidatingMutation(
		({
			id,
			name,
			colour,
			replyWindowDays,
		}: {
			id: string;
			name: string;
			colour: string;
			replyWindowDays: number | null;
		}) => updateApplicationStatus(id, name, colour, replyWindowDays),
		[keys.statuses, keys.applications.all],
	);
}

export function useDeleteApplicationStatus() {
	return useInvalidatingMutation(deleteApplicationStatus, [
		keys.statuses,
		keys.applications.all,
	]);
}

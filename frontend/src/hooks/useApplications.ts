import {
	createQuery,
	keepPreviousData,
	queryOptions,
} from "@tanstack/solid-query";
import {
	clearChase,
	createApplication,
	deleteApplications,
	fetchApplications,
	setChase,
	updateApplication,
} from "../api/applications";
import { keys } from "../api/keys";
import { cvStage, isCvStageBusy } from "../lib/cvStage";
import type { UpdateApplicationPayload } from "../types/application";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export const applicationsQueryOptions = (statusId?: string, chase = false) =>
	queryOptions({
		queryKey: keys.applications.list(statusId, chase),
		queryFn: () => fetchApplications(statusId, chase),
	});

export const chasesQueryOptions = applicationsQueryOptions(undefined, true);

export function useChases() {
	return createQuery(() => chasesQueryOptions);
}

export function useApplications(
	statusId?: () => string | undefined,
	chase?: () => boolean,
) {
	return createQuery(() => ({
		...applicationsQueryOptions(statusId?.(), chase?.() ?? false),
		placeholderData: keepPreviousData,
		refetchInterval: (query) =>
			query.state.data?.some((a) => isCvStageBusy(cvStage(a.Draft)))
				? 3000
				: false,
	}));
}

export function useCreateApplication() {
	return useInvalidatingMutation(createApplication, [
		keys.applications.all,
		keys.jobs.all,
		keys.tasks.all,
	]);
}

export function useUpdateApplication() {
	return useInvalidatingMutation(
		({ id, data }: { id: string; data: UpdateApplicationPayload }) =>
			updateApplication(id, data),
		[keys.applications.all, keys.jobs.all, keys.tasks.all],
	);
}

export function useDeleteApplications() {
	return useInvalidatingMutation(deleteApplications, [
		keys.applications.all,
		keys.jobs.all,
	]);
}

export function useSetChase() {
	return useInvalidatingMutation(
		({ id, chaseBy }: { id: string; chaseBy: string }) => setChase(id, chaseBy),
		[keys.applications.all],
	);
}

export function useClearChase() {
	return useInvalidatingMutation(clearChase, [keys.applications.all]);
}

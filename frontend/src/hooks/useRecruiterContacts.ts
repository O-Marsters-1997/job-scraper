import {
	createQuery,
	keepPreviousData,
	queryOptions,
} from "@tanstack/solid-query";
import { keys } from "../api/keys";
import {
	fetchRecruiterContactSummaries,
	fetchRecruiterContacts,
	logRecruiterContact,
} from "../api/recruiterContacts";
import type { LogRecruiterContactPayload } from "../types/recruiterContact";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export const recruiterContactsQueryOptions = (jobId: string) =>
	queryOptions({
		queryKey: keys.recruiterContacts(jobId),
		queryFn: () => fetchRecruiterContacts(jobId),
	});

export function useRecruiterContacts(jobId: () => string) {
	return createQuery(() => recruiterContactsQueryOptions(jobId()));
}

export function useLogRecruiterContact(jobId: () => string) {
	return useInvalidatingMutation(
		(payload: LogRecruiterContactPayload) =>
			logRecruiterContact(jobId(), payload),
		(contact) => [
			keys.recruiterContacts(contact.job_id),
			keys.recruiterContactSummaries.all,
		],
	);
}

export function useRecruiterContactSummaries(jobIds: () => string[]) {
	return createQuery(() => ({
		queryKey: keys.recruiterContactSummaries.for(jobIds()),
		queryFn: () => fetchRecruiterContactSummaries(jobIds()),
		enabled: jobIds().length > 0,
		placeholderData: keepPreviousData,
	}));
}

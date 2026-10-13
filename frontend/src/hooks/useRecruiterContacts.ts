import { createQuery, queryOptions } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import {
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
		[keys.recruiterContacts(jobId())],
	);
}

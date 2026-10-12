import type {
	LogRecruiterContactPayload,
	RecruiterContact,
	RecruiterContactSummary,
} from "../types/recruiterContact";

export async function logRecruiterContact(
	_payload: LogRecruiterContactPayload,
): Promise<RecruiterContact> {
	throw new Error("not implemented");
}

export async function fetchRecruiterContacts(
	_jobId: string,
): Promise<RecruiterContact[]> {
	throw new Error("not implemented");
}

export async function fetchRecruiterContactsForJobs(
	_jobIds: string[],
): Promise<Record<string, RecruiterContactSummary>> {
	throw new Error("not implemented");
}

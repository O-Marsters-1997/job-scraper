import {
	type LogRecruiterContactPayload,
	type RecruiterContact,
	recruiterContactSchema,
} from "../types/recruiterContact";
import { apiFetch, jsonInit } from "./client";
import { mocked } from "./config";

export async function fetchRecruiterContacts(
	jobId: string,
): Promise<RecruiterContact[]> {
	return mocked(
		(db) => db.getRecruiterContacts(jobId),
		() =>
			apiFetch(
				`/jobs/${encodeURIComponent(jobId)}/recruiter-contacts`,
				recruiterContactSchema.array(),
			),
	);
}

export async function logRecruiterContact(
	jobId: string,
	payload: LogRecruiterContactPayload,
): Promise<RecruiterContact> {
	return mocked(
		(db) => db.logRecruiterContact(jobId, payload),
		() =>
			apiFetch(
				`/jobs/${encodeURIComponent(jobId)}/recruiter-contacts`,
				recruiterContactSchema,
				jsonInit("POST", payload),
			),
	);
}

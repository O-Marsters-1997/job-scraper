import { z } from "zod";
import {
	type LogRecruiterContactPayload,
	type RecruiterContact,
	type RecruiterContactSummary,
	recruiterContactSchema,
	recruiterContactSummarySchema,
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

export async function fetchRecruiterContactSummaries(
	jobIds: string[],
): Promise<Record<string, RecruiterContactSummary>> {
	return mocked(
		(db) => db.getRecruiterContactSummaries(jobIds),
		() =>
			apiFetch(
				`/recruiter-contacts/summaries?job_ids=${encodeURIComponent(jobIds.join(","))}`,
				z.record(z.string(), recruiterContactSummarySchema),
			),
	);
}

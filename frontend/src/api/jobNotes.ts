import { type JobNote, jobNoteSchema } from "../types/jobNote";
import { apiFetch, jsonInit } from "./client";
import { mocked } from "./config";

export async function fetchJobNote(jobId: string): Promise<JobNote> {
	return mocked(
		(db) => db.getJobNote(jobId),
		() => apiFetch(`/jobs/${encodeURIComponent(jobId)}/note`, jobNoteSchema),
	);
}

export async function saveJobNote(
	jobId: string,
	body: string,
): Promise<JobNote> {
	return mocked(
		(db) => db.saveJobNote(jobId, body),
		() =>
			apiFetch(
				`/jobs/${encodeURIComponent(jobId)}/note`,
				jobNoteSchema,
				jsonInit("PUT", { body }),
			),
	);
}

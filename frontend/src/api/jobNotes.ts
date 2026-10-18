import { type JobNote, jobNoteSchema } from "../types/jobNote";
import { apiFetch, jsonInit } from "./client";
import { mocked } from "./config";

export async function fetchJobNotes(jobId: string): Promise<JobNote[]> {
	return mocked(
		(db) => db.getJobNotes(jobId),
		() =>
			apiFetch(
				`/jobs/${encodeURIComponent(jobId)}/notes`,
				jobNoteSchema.array(),
			),
	);
}

export async function createJobNote(
	jobId: string,
	body: string,
): Promise<JobNote> {
	return mocked(
		(db) => db.createJobNote(jobId, body),
		() =>
			apiFetch(
				`/jobs/${encodeURIComponent(jobId)}/notes`,
				jobNoteSchema,
				jsonInit("POST", { body }),
			),
	);
}

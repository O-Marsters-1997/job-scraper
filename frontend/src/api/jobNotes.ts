import { type JobNote, jobNoteSchema, type NoteQuote } from "../types/jobNote";
import { apiFetch, apiFetchVoid, jsonInit } from "./client";
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
	quote: NoteQuote | null,
): Promise<JobNote> {
	return mocked(
		(db) => db.createJobNote(jobId, body, quote),
		() =>
			apiFetch(
				`/jobs/${encodeURIComponent(jobId)}/notes`,
				jobNoteSchema,
				jsonInit("POST", { body, quote }),
			),
	);
}

export async function updateJobNote(
	id: string,
	body: string,
): Promise<JobNote> {
	return mocked(
		(db) => db.updateJobNote(id, body),
		() =>
			apiFetch(
				`/notes/${encodeURIComponent(id)}`,
				jobNoteSchema,
				jsonInit("PATCH", { body }),
			),
	);
}

export async function deleteJobNote(id: string): Promise<void> {
	return mocked(
		(db) => db.deleteJobNote(id),
		() =>
			apiFetchVoid(`/notes/${encodeURIComponent(id)}`, { method: "DELETE" }),
	);
}

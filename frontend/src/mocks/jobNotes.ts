import type { JobNote } from "@/types/jobNote";

const notes = new Map<string, string>();

export function getJobNote(jobId: string): JobNote {
	return { JobID: jobId, Body: notes.get(jobId) ?? "" };
}

export function saveJobNote(jobId: string, body: string): JobNote {
	notes.set(jobId, body);
	return getJobNote(jobId);
}

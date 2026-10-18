import type { JobNote } from "@/types/jobNote";

let notes: JobNote[] = [];

export function getJobNotes(jobId: string): JobNote[] {
	return notes.filter((n) => n.JobID === jobId).reverse();
}

export function createJobNote(jobId: string, body: string): JobNote {
	const now = new Date().toISOString();
	const note: JobNote = {
		ID: `note-${notes.length + 1}`,
		JobID: jobId,
		Body: body,
		CreatedAt: now,
		UpdatedAt: now,
	};
	notes = [...notes, note];
	return note;
}

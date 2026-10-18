import type { JobNote, NoteQuote } from "@/types/jobNote";

let notes: JobNote[] = [];
let nextId = 1;

export function getJobNotes(jobId: string): JobNote[] {
	return notes.filter((n) => n.JobID === jobId).reverse();
}

export function createJobNote(
	jobId: string,
	body: string,
	quote: NoteQuote | null,
): JobNote {
	const now = new Date().toISOString();
	const note: JobNote = {
		ID: `note-${nextId++}`,
		JobID: jobId,
		Body: body,
		Quote: quote,
		CreatedAt: now,
		UpdatedAt: now,
	};
	notes = [...notes, note];
	return note;
}

export function updateJobNote(id: string, body: string): JobNote {
	const note = notes.find((n) => n.ID === id);
	if (!note) throw new Error("note not found");
	const updated = { ...note, Body: body, UpdatedAt: new Date().toISOString() };
	notes = notes.map((n) => (n.ID === id ? updated : n));
	return updated;
}

export function deleteJobNote(id: string): void {
	notes = notes.filter((n) => n.ID !== id);
}

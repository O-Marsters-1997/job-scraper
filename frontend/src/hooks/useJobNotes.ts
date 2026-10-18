import { createQuery } from "@tanstack/solid-query";
import { createJobNote, fetchJobNotes } from "../api/jobNotes";
import { keys } from "../api/keys";
import type { NoteQuote } from "../types/jobNote";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export function useJobNotes(jobId: () => string) {
	return createQuery(() => ({
		queryKey: keys.jobNotes(jobId()),
		queryFn: () => fetchJobNotes(jobId()),
	}));
}

export function useCreateJobNote(jobId: () => string) {
	return useInvalidatingMutation(
		(note: { body: string; quote: NoteQuote | null }) =>
			createJobNote(jobId(), note.body, note.quote),
		(note) => [keys.jobNotes(note.JobID)],
	);
}

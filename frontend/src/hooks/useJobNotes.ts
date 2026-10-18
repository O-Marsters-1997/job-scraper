import { createQuery } from "@tanstack/solid-query";
import {
	createJobNote,
	deleteJobNote,
	fetchJobNotes,
	updateJobNote,
} from "../api/jobNotes";
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

export function useUpdateJobNote(jobId: () => string) {
	return useInvalidatingMutation(
		(v: { id: string; body: string }) => updateJobNote(v.id, v.body),
		() => [keys.jobNotes(jobId())],
	);
}

export function useDeleteJobNote(jobId: () => string) {
	return useInvalidatingMutation(
		(id: string) => deleteJobNote(id),
		() => [keys.jobNotes(jobId())],
	);
}

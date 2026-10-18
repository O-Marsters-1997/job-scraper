import { createQuery } from "@tanstack/solid-query";
import { createJobNote, fetchJobNotes } from "../api/jobNotes";
import { keys } from "../api/keys";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export function useJobNotes(jobId: () => string) {
	return createQuery(() => ({
		queryKey: keys.jobNotes(jobId()),
		queryFn: () => fetchJobNotes(jobId()),
	}));
}

export function useCreateJobNote(jobId: () => string) {
	return useInvalidatingMutation(
		(body: string) => createJobNote(jobId(), body),
		(note) => [keys.jobNotes(note.JobID)],
	);
}

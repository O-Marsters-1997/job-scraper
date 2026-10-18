import { createQuery, queryOptions } from "@tanstack/solid-query";
import {
	createJobNote,
	deleteJobNote,
	fetchJobNotes,
	updateJobNote,
} from "../api/jobNotes";
import { keys } from "../api/keys";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export const jobNotesQueryOptions = (jobId: string) =>
	queryOptions({
		queryKey: keys.jobNotes(jobId),
		queryFn: () => fetchJobNotes(jobId),
	});

export function useJobNotes(jobId: () => string) {
	return createQuery(() => jobNotesQueryOptions(jobId()));
}

export function useCreateJobNote(jobId: () => string) {
	return useInvalidatingMutation(
		(body: string) => createJobNote(jobId(), body),
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
